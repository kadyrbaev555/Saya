# SAYA Flutter

Проверенная среда: Flutter 3.50.0, Dart 3.14.0. Зависимости закреплены и pubspec.lock хранится в Git. Material 3 используется через официальный material_ui package после выделения его из Flutter SDK.

```sh
flutter pub get
flutter analyze
flutter test
flutter build web
flutter run -d chrome --web-port=3000
SAYA_REALTIME_TEST=1 flutter test test/realtime_integration_test.dart
```

Интеграционный тест требует локальный Compose и Docker CLI: получает реальный token, подписывается через Centrifugo SDK, проверяет demo и assessment.completed, затем читает REST. Обычные тесты проверяют auth states и UI path login→consent→survey→result на fake API. Realtime SDK обновляет connection/subscription JWT, reconnect; pending REST polling 2 секунды.

Web refresh только HttpOnly cookie; native — OS secure storage. Долгоживущие токены не пишутся в browser localStorage или preferences. «Продолжить сессию» добровольно восстанавливает её через refresh; logout обращается к серверу. API refresh singleflight, HTTP timeout прерывает запрос, provider dispose освобождает соединения.

API_BASE_URL и WS_URL конфигурируются --dart-define; это адреса, не backend secrets. Android emulator: 10.0.2.2 вместо localhost. Android/macOS runners и entitlements созданы; native builds требуют Android SDK/полного Xcode/CocoaPods и здесь не проверены.
