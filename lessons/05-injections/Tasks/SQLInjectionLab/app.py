import os

import psycopg
from psycopg.rows import dict_row
from flask import Flask, redirect, render_template, request, session, url_for


app = Flask(__name__)
app.config.update(
    SECRET_KEY=os.environ.get("FLASK_SECRET_KEY", "local-training-key-change-me"),
    SESSION_COOKIE_HTTPONLY=True,
    SESSION_COOKIE_SAMESITE="Lax",
)

DATABASE_URL = os.environ.get(
    "DATABASE_URL", "postgresql://ctfapp:ctfapp_dev_only@db:5432/ctf_lab"
)


def get_connection():
    return psycopg.connect(DATABASE_URL, row_factory=dict_row)


@app.after_request
def add_response_headers(response):
    response.headers["Cache-Control"] = "no-store"
    response.headers["X-Content-Type-Options"] = "nosniff"
    return response


@app.route("/login", methods=["GET", "POST"])
def login():
    if request.method == "GET":
        return render_template("login.html")

    login_value = request.form.get("login", "")
    password_value = request.form.get("password", "")

    # ВАЖНО: это намеренная SQL-инъекция для изолированной CTF-лаборатории.
    # В production- коде значения нужно передавать параметрами запроса.
    query = (
        "SELECT id, login, display_name, private_note "
        "FROM users "
        f"WHERE login = '{login_value}' AND password = '{password_value}'"
    )

    try:
        with get_connection() as connection, connection.cursor() as cursor:
            cursor.execute(query)
            user = cursor.fetchone()
    except psycopg.Error:
        # Ошибки БД намеренно не показываются пользователю: студентам доступен
        # исходный код и журналы контейнера преподавателя.
        user = None

    if user is None:
        return render_template(
            "login.html", error="Неверный логин или пароль."
        ), 401

    session.clear()
    session["user_id"] = user["id"]
    return redirect(url_for("profile"))


@app.route("/")
def profile():
    user_id = session.get("user_id")
    if user_id is None:
        return redirect(url_for("login"))

    # Пример корректного доступа к БД: запрос параметризован.
    with get_connection() as connection, connection.cursor() as cursor:
        cursor.execute(
            "SELECT login, display_name, private_note FROM users WHERE id = %s",
            (user_id,),
        )
        user = cursor.fetchone()

    if user is None:
        session.clear()
        return redirect(url_for("login"))

    return render_template("profile.html", user=user)


@app.post("/logout")
def logout():
    session.clear()
    return redirect(url_for("login"))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
