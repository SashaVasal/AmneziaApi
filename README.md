# amnezia_go

Единственный поддерживаемый способ запуска проекта:

- Amnezia контейнер (`amnezia-awg`) уже создан приложением Amnezia.
- Этот проект поднимает только API-менеджер и подключается к сетевому namespace контейнера `amnezia-awg`.

## Требования

- Linux сервер
- запущенный контейнер `amnezia-awg`
- интерфейс `wg0` внутри `amnezia-awg`:

```bash
docker exec amnezia-awg ip link show wg0
```

## Настройка `.env`

Минимум проверь:

```env
AWG_EXTERNAL_INTERFACE=wg0
AWG_RUNTIME_MODE=strict
AWG_SERVER_HOST=<PUBLIC_IP_OR_DOMAIN>
AWG_SERVER_PORT=48393
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<STRONG_PASSWORD>
JWT_SECRET=<LONG_RANDOM_SECRET>
```

## Запуск

```bash
docker compose up -d --build
```

## Проверка

```bash
docker compose logs api --tail=100
```

В логе API не должно быть ошибок инициализации WG интерфейса.

## API

- `POST /api/auth/login`
- `GET /api/peers`
- `POST /api/peers`
- `DELETE /api/peers/:id`
- `GET /api/peers/:id/config`
- `GET /api/stats`
