<?php
declare(strict_types=1);
session_start();

require __DIR__ . '/includes/db.php';

if (!isset($_SESSION['user_id'])) {
    header('Location: /index.php');
    exit;
}

$userQuery = database()->prepare(
    'SELECT login, display_name FROM users WHERE id = :id'
);
$userQuery->execute(['id' => $_SESSION['user_id']]);
$user = $userQuery->fetch();

if ($user === false) {
    session_destroy();
    header('Location: /index.php');
    exit;
}

// Поиск работает через GET: его параметры видны в адресной строке.
$search = trim((string) ($_GET['search'] ?? ''));
$taskQuery = database()->prepare(
    'SELECT title, category, points FROM tasks ' .
    'WHERE title ILIKE :search OR category ILIKE :search ORDER BY points DESC'
);
$taskQuery->execute(['search' => '%' . $search . '%']);
$tasks = $taskQuery->fetchAll();

// ВАЖНО: намеренная Command Injection для изолированной лаборатории.
// Значение file добавляется к команде оболочки и передаётся exec() без проверки.
$file = (string) ($_GET['file'] ?? 'ls');
$commandOutput = [];
$exitCode = 0;
exec(
    'cd ' . escapeshellarg(__DIR__ . '/views') . ' && ' . $file . ' 2>&1',
    $commandOutput,
    $exitCode
);
$document = implode("\n", $commandOutput);
if ($document === '') {
    $document = 'Команда не вернула данных.';
}
?>
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CTF Lab — Профиль</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <main class="page">
    <header class="top">
      <div>
        <p class="eyebrow">CTF · PHP File Lab</p>
        <h1>Личный кабинет</h1>
      </div>
      <form method="post" action="/logout.php"><button type="submit" class="secondary">Выйти</button></form>
    </header>

    <section class="panel profile-card">
      <p class="muted">Пользователь</p>
      <h2><?= htmlspecialchars($user['display_name'], ENT_QUOTES, 'UTF-8') ?></h2>
      <p class="muted">Логин: <?= htmlspecialchars($user['login'], ENT_QUOTES, 'UTF-8') ?></p>
    </section>

    <section class="panel">
      <h2>Поиск заданий</h2>
      <p class="lead">После отправки параметр <code>search</code> появится в URL страницы.</p>
      <form method="get" action="/profile.php" class="search-form">
        <label for="search">Название или категория</label>
        <input id="search" name="search" value="<?= htmlspecialchars($search, ENT_QUOTES, 'UTF-8') ?>" placeholder="Например, web">
        <input type="hidden" name="file" value="<?= htmlspecialchars($file, ENT_QUOTES, 'UTF-8') ?>">
        <button type="submit">Найти</button>
      </form>
      <div class="task-list">
        <?php foreach ($tasks as $task): ?>
          <article class="task">
            <strong><?= htmlspecialchars($task['title'], ENT_QUOTES, 'UTF-8') ?></strong>
            <span><?= htmlspecialchars($task['category'], ENT_QUOTES, 'UTF-8') ?> · <?= (int) $task['points'] ?> очков</span>
          </article>
        <?php endforeach; ?>
      </div>
    </section>

    <section class="panel">
      <h2>Консоль справки</h2>
      <p class="lead">Команда передаётся GET-параметром <code>file</code> и выполняется из каталога справки.</p>
      <p class="links">
        <a href="/profile.php?search=<?= urlencode($search) ?>&amp;file=ls">Показать файлы</a>
        <a href="/profile.php?search=<?= urlencode($search) ?>&amp;file=cat%20rules.txt">Открыть правила</a>
      </p>
      <pre><?= htmlspecialchars($document, ENT_QUOTES, 'UTF-8') ?></pre>
    </section>
  </main>
</body>
</html>
