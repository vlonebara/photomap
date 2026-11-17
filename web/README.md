# PhotoMap

Фотокарта на Vue, TypeScript, Tailwind CSS и MapLibre. OpenFreeMap отдаёт базовую карту. Go API отвечает за снимки, серии, сессии, загрузки и модерацию.

## Локальный запуск

1. Запусти сервер из каталога `server` актуального `main`. Ему нужны инициализированная `photomap.db`, каталог `media` и переменная `JWT_SECRET` со значением не короче 32 байт.

   ```sh
   export JWT_SECRET="$(openssl rand -hex 32)"
   go run ./cmd
   ```

   Сервер слушает порт `8080`. Сохраняй одно значение `JWT_SECRET` между запусками, если хочешь сохранить действующие сессии.

2. В каталоге `web` выполни:

   ```sh
   npm ci
   npm run dev
   ```

   Открой адрес, который покажет Vite. Он перенаправляет `/api` на `http://localhost:8080`, сохраняя заголовок `Host`. Если сервер слушает по другому адресу, задай `PHOTOMAP_API_TARGET` перед запуском Vite.

Для проверки собранного фронтенда локально используй `npm run build` и затем `npm run preview`. Preview тоже проксирует `/api`.

## Публичный запуск

Выполни `npm ci` и `npm run build`. Раздавай содержимое `dist` по HTTPS и направляй `/api/` с того же домена на Go. Для запуска Go используй отдельную базу без демонстрационных аккаунтов, постоянный закрытый `JWT_SECRET` и `COOKIE_SECURE=true`. Подготовь отдельную учётную запись модератора, чтобы принимать снимки из очереди. Каталог `media` доступен только процессу Go и через его маршруты.

Пример для существующего HTTPS блока Nginx. Укажи фактический каталог сборки и адрес Go:

```nginx
root /var/www/photomap;

location / {
    try_files $uri $uri/ /index.html;
}

location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $http_host;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_read_timeout 60s;
    client_max_body_size 11m;
}
```

Так браузер использует одну сессию и для API, и для закрытых изображений. Отключение буферизации позволяет доставлять события карты сразу. Для базовой карты нужен доступ к OpenFreeMap.

[Контракт Go API](API.md)
