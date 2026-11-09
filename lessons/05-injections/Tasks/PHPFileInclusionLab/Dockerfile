FROM php:8.4-apache

RUN apt-get update \
    && apt-get install -y --no-install-recommends libpq-dev \
    && docker-php-ext-install pdo_pgsql \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /var/www/html

COPY --chown=www-data:www-data . /var/www/html
RUN mkdir -p /var/www/private \
    && cp /var/www/html/private/flag.txt /var/www/private/flag.txt \
    && rm -rf /var/www/html/private \
    && a2enmod headers

EXPOSE 80
