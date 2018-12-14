# Протокол общения фронта и бека

- Таймаут на подключение по ВС 10 сек

```javascript
{
    "status": "connected" // успешно подключен
}
```

или

```javascript
{
    "status": "playing" // этот аккаунт уже в игре
}
```

- Старт игры

```javascript
{
    "status": "started",
    "payload": {
        "opponentId": 50, // тут мы делаем GET /profile?id=50 и рисуем ник, аву
        "playerNum": 1, // в игровых стейтах будет player1 и player2, тут нам приходит наш номер
        "stateConst": {
            "gameTime": 30 // время игры
        }
    }
}
```

- Стейты

```javascript
{
    "status": "state",
    "payload": {
        "player1": {
            "score": 10,
            "percentsX": 50, // 0-100
            "percentsY": 10, // 0-100
            "targetList": [1, 2, 6, 3] // 1-6
        },
        "player2": {
            "score": 15,
            "percentsX": 50, // 0-100
            "percentsY": 10, // 0-100
            "targetList": [1, 2, 6, 3] // 1-6
        },
        "products": [
            {
                "percentsX": 50, // 0-100
                "percentsY": 10, // 0-100
                "type": 2 // 1-6
            },
            ...
        ],
        "collected": [
            {
                "percentsX": 50, // 0-100
                "percentsY": 10, // 0-100
                "playerNum": 1, // 1 или 2 кто собрал
                "points": -1 // int очки за собранный продукт
            },
            ...
        ]
    }
}
```

- Окончание игры

```javascript
{
    "status": "disconnected" // "time_over"
}
```

- ОТ ФРОНТА:

```javascript
{
    "actions": 110 // битовая маска действий: left, jump, right
}
```

- Условие победы:

1) время вышло и у тебя больше очков
2) ничья если очков одинаково
3) ничья для аутистов (если у обоих отрицательные очки)
