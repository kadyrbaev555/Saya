# Проверка macOS и Android

На 2026-11-14: Flutter 3.50.0 установлен в /tmp/saya-flutter-sdk, macOS 14.6.1 ARM64.
Полный Xcode, CocoaPods и Android SDK не обнаружены. Native UI/E2E ещё не проверены.

## macOS

Установите совместимый с вашей macOS полный Xcode (при необходимости обновите macOS),
откройте его, установите предлагаемые компоненты и примите лицензию.
Выберите developer directory:

```sh
sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer
sudo xcodebuild -runFirstLaunch
```

Для native plugins требуется настройка согласно flutter doctor; CocoaPods сейчас отсутствует.
Официальная инструкция: https://docs.flutter.dev/platform-integration/macos/setup

Из корня репозитория:

```sh
make dev-up
make FLUTTER=/tmp/saya-flutter-sdk/bin/flutter frontend-macos
```

## Android

Установите Android Studio для Apple Silicon:
https://developer.android.com/studio
Пройдите Setup Wizard. В SDK Manager установите Android SDK, Platform-Tools,
Build-Tools, Command-line Tools и Android Emulator.
В Device Manager создайте телефон с ARM64 system image и запустите его.
Сверьте установку с https://docs.flutter.dev/platform-integration/android/setup

```sh
/tmp/saya-flutter-sdk/bin/flutter doctor --android-licenses
/tmp/saya-flutter-sdk/bin/flutter doctor -v
/tmp/saya-flutter-sdk/bin/flutter devices
make FLUTTER=/tmp/saya-flutter-sdk/bin/flutter frontend-android ANDROID_DEVICE=emulator-5554
```

ID устройства берите из flutter devices. Для эмулятора адрес Mac — 10.0.2.2:
API http://10.0.2.2:8080, WebSocket ws://10.0.2.2:8000/connection/websocket.
Makefile передаёт эти адреса автоматически.
Debug Android разрешает cleartext только для 10.0.2.2/localhost/127.0.0.1;
release не получает debug network security config и требует HTTPS/WSS.
Физическое устройство требует отдельной настройки адресов или adb reverse.
Flutter находится во временном каталоге: его нужно перенести/переустановить для постоянной работы.

## Ручной сценарий на каждой платформе

1. Создать отдельный тестовый аккаунт (@example.test); проверить профиль.
2. Выйти и войти повторно.
3. Пройти демонстрационный опрос с согласием на обработку; внешний LLM consent не включать.
4. Проверить переход ожидание → результат и отсутствие чувствительных данных в уведомлении.
5. Перезапустить приложение и проверить восстановление сессии через защищённое хранилище.
6. Отключить сеть и проверить обработку ошибки/reconnect; затем повторить запрос.
7. Выйти, перезапустить и проверить отсутствие авторизованной сессии.

Проверка Keychain/Android secure storage и настоящих native WebSocket требует
запущенного приложения на соответствующей платформе; widget-тесты её не заменяют.
