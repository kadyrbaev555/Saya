import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class Vault {
  final _storage = const FlutterSecureStorage();
  Future<String?> read() => _storage.read(key: 'saya_refresh');
  Future<void> write(String? value) =>
      value == null
          ? _storage.delete(key: 'saya_refresh')
          : _storage.write(key: 'saya_refresh', value: value);
}
