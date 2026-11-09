<?php
declare(strict_types=1);

function database(): PDO
{
    static $connection = null;

    if ($connection instanceof PDO) {
        return $connection;
    }

    $connection = new PDO(
        getenv('DB_DSN') ?: 'pgsql:host=db;port=5432;dbname=php_injection_lab',
        getenv('DB_USER') ?: 'ctfapp',
        getenv('DB_PASSWORD') ?: 'ctfapp_dev_only',
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );

    return $connection;
}
