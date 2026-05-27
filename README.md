# 2026_1_VKernel_Team

Репозиторий команды VKernel_Team\
Проект – Юла\
Наше название - Клевер\
Рабочий процесс – [github-flow](https://docs.github.com/en/get-started/using-github/github-flow)

## Участники

- [Николай Угрюмов](https://github.com/ugryum1)
- [Егор Воробьев](https://github.com/virsi)
- [Мантров Леонид](https://github.com/leonidmantrov)
- [Минбулатов Абдулвагаб](https://github.com/PrincepsMontis)

## Ссылки

- [Production](https://clover-go.ru)
- [Swagger](https://clover-go.ru/swagger)
- [Figma](https://www.figma.com/design/xbY95p954ybj8mmFsK63KU/%D0%A0%D0%9A1-%D0%A4%D0%B8%D0%B3%D0%BC%D0%B0?node-id=0-1&p=f)
- [Frontend](https://github.com/frontend-park-mail-ru/2026_1_VKernel_Team)
- [Требования](https://docs.google.com/spreadsheets/d/1h1QaRvRbF2eBUzdV1tLU62hS68bdNH-bLht6N29uDK8/edit?gid=1085759601#gid=1085759601&range=A11)

## How to run

Требуется Docker и `make`.

```bash
cp .env.example .env   # заполнить значения
make run
```

Сервис поднимется через docker compose. Остановка — `make stop`, логи — `make logs`.
