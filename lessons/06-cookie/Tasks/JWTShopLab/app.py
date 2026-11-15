import base64
import hashlib
import hmac
import json
import os
import sqlite3
from http import cookies
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs


APP_DIR = Path(__file__).resolve().parent
DB_PATH = Path(os.environ.get("DB_PATH", APP_DIR / "shop.db"))
JWT_SECRET = "andres"
INITIAL_BALANCE = 10_000
FLAG_PRICE = 1_000_000
FLAG = "CTF{weak_jwt_secret_allows_identity_forgery}"

PRODUCTS = [
    {"id": "sticker", "name": "CTF-стикерпак", "description": "Набор наклеек для ноутбука.", "price": 50, "flag": ""},
    {"id": "guide", "name": "Памятка участника", "description": "Краткий чек-лист перед соревнованием.", "price": 250, "flag": ""},
    {"id": "hoodie", "name": "Худи факультатива", "description": "Лимитированный мерч команды.", "price": 3500, "flag": ""},
    {"id": "flag", "name": "Секретный флаг", "description": "Эксклюзивный цифровой товар для VIP-покупателей.", "price": FLAG_PRICE, "flag": FLAG},
]


def db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn


def init_db():
    with db() as conn:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS users (
                login TEXT PRIMARY KEY,
                password TEXT NOT NULL,
                balance INTEGER NOT NULL
            )
            """
        )
        conn.execute(
            "INSERT OR IGNORE INTO users(login, password, balance) VALUES (?, ?, ?)",
            ("rich", "richpass", 5_000_000),
        )


def b64url_encode(data):
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def b64url_decode(data):
    padding = "=" * (-len(data) % 4)
    return base64.urlsafe_b64decode(data + padding)


def create_jwt(login):
    header = {"alg": "HS256", "typ": "JWT"}
    payload = {"login": login}
    header_part = b64url_encode(json.dumps(header, separators=(",", ":")).encode())
    payload_part = b64url_encode(json.dumps(payload, separators=(",", ":")).encode())
    signing_input = f"{header_part}.{payload_part}"
    signature = hmac.new(JWT_SECRET.encode(), signing_input.encode(), hashlib.sha256).digest()
    return f"{signing_input}.{b64url_encode(signature)}"


def verify_jwt(token):
    try:
        header_part, payload_part, signature_part = token.split(".")
        signing_input = f"{header_part}.{payload_part}"
        expected = hmac.new(JWT_SECRET.encode(), signing_input.encode(), hashlib.sha256).digest()
        actual = b64url_decode(signature_part)
        if not hmac.compare_digest(expected, actual):
            return None
        payload = json.loads(b64url_decode(payload_part))
        login = payload.get("login")
        if not isinstance(login, str) or not login:
            return None
        return login
    except Exception:
        return None


def get_user(login):
    with db() as conn:
        return conn.execute("SELECT login, balance FROM users WHERE login = ?", (login,)).fetchone()


def find_user(login, password):
    with db() as conn:
        return conn.execute(
            "SELECT login FROM users WHERE login = ? AND password = ?",
            (login, password),
        ).fetchone()


def create_user(login, password):
    try:
        with db() as conn:
            conn.execute(
                "INSERT INTO users(login, password, balance) VALUES (?, ?, ?)",
                (login, password, INITIAL_BALANCE),
            )
        return True
    except sqlite3.IntegrityError:
        return False


def list_users():
    with db() as conn:
        return conn.execute("SELECT login, balance FROM users ORDER BY balance DESC, login ASC").fetchall()


def change_balance(login, new_balance):
    with db() as conn:
        conn.execute("UPDATE users SET balance = ? WHERE login = ?", (new_balance, login))


def render_template(name, **context):
    template = (APP_DIR / "templates" / name).read_text(encoding="utf-8")
    for key, value in context.items():
        template = template.replace("{{" + key + "}}", str(value))
    return template.encode("utf-8")


def render_products():
    cards = []
    for product in PRODUCTS:
        cards.append(
            f"""
            <article class="panel product">
              <p class="tag">{product["id"]}</p>
              <h2>{product["name"]}</h2>
              <p class="lead">{product["description"]}</p>
              <div class="product-bottom">
                <strong>${product["price"]}</strong>
                <form method="post" action="/buy">
                  <input type="hidden" name="product" value="{product["id"]}">
                  <button type="submit">Купить</button>
                </form>
              </div>
            </article>
            """
        )
    return "\n".join(cards)


def render_users(rows):
    items = []
    for row in rows:
        items.append(f"<tr><td>{row['login']}</td><td>${row['balance']}</td></tr>")
    return "\n".join(items)


def notice(kind, text):
    if not text:
        return ""
    return f'<p class="notice {kind}" role="alert">{text}</p>'


class ShopHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/static/"):
            return self.static_file()
        if self.path == "/":
            return self.store()
        if self.path == "/login":
            return self.login_form()
        if self.path == "/signup":
            return self.signup_form()
        if self.path == "/users":
            return self.users()
        self.send_error(404)

    def do_POST(self):
        if self.path == "/login":
            return self.login()
        if self.path == "/signup":
            return self.signup()
        if self.path == "/buy":
            return self.buy()
        if self.path == "/logout":
            return self.logout()
        self.send_error(404)

    def read_form(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length).decode("utf-8")
        return {key: values[0] for key, values in parse_qs(body).items()}

    def current_login(self):
        raw_cookie = self.headers.get("Cookie", "")
        jar = cookies.SimpleCookie(raw_cookie)
        morsel = jar.get("session")
        if not morsel:
            return None
        return verify_jwt(morsel.value)

    def current_user(self):
        login = self.current_login()
        if not login:
            return None
        return get_user(login)

    def send_html(self, body, status=200):
        self.send_response(status)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.end_headers()
        self.wfile.write(body)

    def redirect(self, location):
        self.send_response(303)
        self.send_header("Location", location)
        self.end_headers()

    def set_session(self, login):
        self.send_response(303)
        self.send_header("Location", "/")
        self.send_header("Set-Cookie", f"session={create_jwt(login)}; Path=/; SameSite=Lax")
        self.end_headers()

    def store(self, error="", success="", receipt=""):
        user = self.current_user()
        if not user:
            return self.redirect("/login")
        body = render_template(
            "store.html",
            title="JWT Shop",
            login=user["login"],
            balance=user["balance"],
            products=render_products(),
            error=notice("error", error),
            success=notice("success", success),
            receipt=receipt,
        )
        self.send_html(body)

    def login_form(self, error=""):
        self.send_html(render_template("login.html", title="Вход - JWT Shop", error=notice("error", error)))

    def signup_form(self, error=""):
        self.send_html(render_template("signup.html", title="Регистрация - JWT Shop", error=notice("error", error)))

    def users(self):
        user = self.current_user()
        if not user:
            return self.redirect("/login")
        body = render_template(
            "users.html",
            title="Пользователи - JWT Shop",
            login=user["login"],
            users=render_users(list_users()),
        )
        self.send_html(body)

    def login(self):
        form = self.read_form()
        login = form.get("login", "").strip()
        password = form.get("password", "")
        if not find_user(login, password):
            return self.login_form("Неверный логин или пароль.")
        self.set_session(login)

    def signup(self):
        form = self.read_form()
        login = form.get("login", "").strip()
        password = form.get("password", "")
        if len(login) < 3 or len(password) < 4:
            return self.signup_form("Логин - от 3 символов, пароль - от 4 символов.")
        if not create_user(login, password):
            return self.signup_form("Такой логин уже занят.")
        self.set_session(login)

    def buy(self):
        user = self.current_user()
        if not user:
            return self.redirect("/login")
        form = self.read_form()
        product = next((item for item in PRODUCTS if item["id"] == form.get("product")), None)
        if not product:
            return self.send_error(404)
        if user["balance"] < product["price"]:
            return self.store(error="Недостаточно средств для покупки этого товара.")
        new_balance = user["balance"] - product["price"]
        change_balance(user["login"], new_balance)
        receipt = f"""
        <section class="panel receipt">
          <p class="eyebrow">ПОКУПКА ПОДТВЕРЖДЕНА</p>
          <h2>{product["name"]}</h2>
          {"<code>" + product["flag"] + "</code>" if product["flag"] else "<p>Товар добавлен в вашу коллекцию.</p>"}
        </section>
        """
        self.store(success="Оплата прошла успешно.", receipt=receipt)

    def logout(self):
        self.send_response(303)
        self.send_header("Location", "/login")
        self.send_header("Set-Cookie", "session=; Path=/; Max-Age=0; SameSite=Lax")
        self.end_headers()

    def static_file(self):
        filename = self.path.removeprefix("/static/")
        if filename != "style.css":
            return self.send_error(404)
        data = (APP_DIR / "static" / "style.css").read_bytes()
        self.send_response(200)
        self.send_header("Content-Type", "text/css; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt, *args):
        print("%s - %s" % (self.address_string(), fmt % args))


if __name__ == "__main__":
    init_db()
    port = int(os.environ.get("PORT", "8080"))
    server = ThreadingHTTPServer(("0.0.0.0", port), ShopHandler)
    print(f"JWT Shop Lab listening on :{port}")
    server.serve_forever()
