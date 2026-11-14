import 'dart:convert';
import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import '../security/vault.dart';
import 'transport.dart';

class ApiError implements Exception {
  final int status;
  ApiError(this.status);
  @override
  String toString() =>
      status == 401
          ? 'Сессия завершена'
          : 'Не удалось выполнить запрос ($status)';
}

class Api {
  final http.Client client;
  final Vault vault;
  Api({http.Client? transport, Vault? secureVault})
    : client = transport ?? createClient(),
      vault = secureVault ?? Vault();
  String? access, _csrf;
  Future<void>? _refreshing;
  bool _closed = false;
  static const base = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );
  Future<Map<String, dynamic>> _send(
    String path, {
    Map<String, dynamic>? body,
    bool session = false,
  }) async {
    final headers = <String, String>{
      'Content-Type': 'application/json',
      if (!kIsWeb) 'X-Saya-Client': 'native',
      if (access != null) 'Authorization': 'Bearer $access',
      if (_csrf != null) 'X-CSRF-Token': _csrf!,
    };
    final data = body == null ? null : Map<String, dynamic>.from(body);
    if (session && !kIsWeb) data!['refresh_token'] = await vault.read();
    if (_closed) throw StateError('Client disposed');
    final abort = Completer<void>();
    final timer = Timer(const Duration(seconds: 15), () {
      if (!abort.isCompleted) abort.complete();
    });
    final request = http.AbortableRequest(
      body == null ? 'GET' : 'POST',
      Uri.parse('$base/api/v1$path'),
      abortTrigger: abort.future,
    )..headers.addAll(headers);
    if (data != null) request.body = jsonEncode(data);
    late final http.Response response;
    try {
      response = await http.Response.fromStream(await client.send(request));
    } finally {
      timer.cancel();
    }
    if (_closed) throw StateError('Client disposed');
    if (response.statusCode >= 400) throw ApiError(response.statusCode);
    return response.body.isEmpty
        ? <String, dynamic>{}
        : jsonDecode(response.body) as Map<String, dynamic>;
  }

  Future<void> _accept(Map<String, dynamic> data) async {
    access = data['access_token'] as String?;
    _csrf = data['csrf_token'] as String?;
    if (!kIsWeb) await vault.write(data['refresh_token'] as String?);
  }

  Future<void> authenticate(
    String action,
    String email,
    String password,
  ) async => _accept(
    await _send('/auth/$action', body: {'email': email, 'password': password}),
  );
  Future<void> refresh() async {
    if (_refreshing != null) return _refreshing!;
    final future = _rotate();
    _refreshing = future;
    try {
      await future;
    } finally {
      _refreshing = null;
    }
  }

  Future<void> _rotate() async {
    if (kIsWeb && _csrf == null) {
      _csrf = (await _send('/auth/csrf'))['csrf_token'] as String;
    }
    await _accept(await _send('/auth/refresh', body: {}, session: true));
  }

  Future<Map<String, dynamic>> request(
    String path, {
    Map<String, dynamic>? body,
  }) async {
    try {
      return await _send(path, body: body);
    } on ApiError catch (e) {
      if (e.status != 401) rethrow;
      await refresh();
      return _send(path, body: body);
    }
  }

  Future<void> logout() async {
    if (kIsWeb && _csrf == null) {
      _csrf = (await _send('/auth/csrf'))['csrf_token'] as String;
    }
    await _send('/auth/logout', body: {}, session: true);
    access = null;
    _csrf = null;
    await vault.write(null);
  }

  void close() {
    _closed = true;
    access = null;
    client.close();
  }
}
