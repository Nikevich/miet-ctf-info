<?php
declare(strict_types=1);
session_start();

require __DIR__ . '/includes/db.php';

if (isset($_SESSION['user_id'])) {
    header('Location: /profile.php');
    exit;
}

$error = '';
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $login = (string) ($_POST['login'] ?? '');
    $password = (string) ($_POST['password'] ?? '');

    $statement = database()->prepare(
        'SELECT id FROM users WHERE login = :login AND password = :password'
    );
    $statement->execute(['login' => $login, 'password' => $password]);
    $user = $statement->fetch();

    if ($user === false) {
        $error = 'Неверный логин или пароль.';
    } else {
        session_regenerate_id(true);
        $_SESSION['user_id'] = (int) $user['id'];
        header('Location: /profile.php');
        exit;
    }
}
?>
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CTF Lab — Вход</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body class="centered">
  <main class="panel login-panel">
    <p class="eyebrow">CTF · PHP File Lab</p>
    <h1>Вход в личный кабинет</h1>
    <p class="lead">Учебный сервис факультатива по безопасности.</p>
    <?php if ($error !== ''): ?>
      <p class="error" role="alert"><?= htmlspecialchars($error, ENT_QUOTES, 'UTF-8') ?></p>
    <?php endif; ?>
    <form method="post" action="/index.php" autocomplete="off">
      <label for="login">Логин</label>
      <input id="login" name="login" required autofocus>
      <label for="password">Пароль</label>
      <input id="password" name="password" type="password" required>
      <button type="submit">Войти</button>
    </form>
    <p class="hint">Демо-учётная запись: <code>student / ctf-student-2026</code></p>
  </main>
</body>
</html>
