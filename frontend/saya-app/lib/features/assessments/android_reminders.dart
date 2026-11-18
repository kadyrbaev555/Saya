import 'dart:async';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../../core/network/api.dart';

/// Stores the local opt-in separately from the authenticated API session.
class AndroidReminders {
  static final instance = AndroidReminders();
  static bool get supported =>
      !kIsWeb && defaultTargetPlatform == TargetPlatform.android;
  final enabledFor = ValueNotifier<String?>(null);
  final _storage = const FlutterSecureStorage();
  Future<bool>? _ready;
  String? Function()? _currentOwner;
  Future<void> _queue = Future.value();
  StreamSubscription<String>? _registrations;
  StreamSubscription<RemoteMessage>? _taps;

  Future<bool> initialize(
    Api api,
    String? Function() currentOwner,
    void Function() openSurvey,
  ) {
    _currentOwner = currentOwner;
    return _ready ??= _initialize(api, currentOwner, openSurvey);
  }

  Future<bool> _initialize(
    Api api,
    String? Function() currentOwner,
    void Function() openSurvey,
  ) async {
    if (!supported) return false;
    try {
      await Firebase.initializeApp();
      enabledFor.value = await _storage.read(key: 'weekly_owner');
      final messaging = FirebaseMessaging.instance;
      _registrations = messaging.onRegistered.listen((fid) {
        final owner = currentOwner();
        if (owner != null && owner == enabledFor.value) {
          _queue = _queue
              .then((_) async {
                if (currentOwner() == owner && enabledFor.value == owner)
                  await _bind(api, fid);
              })
              .catchError((Object _) {});
        }
      });
      void onTap(RemoteMessage? m) {
        if (m?.data['route'] == 'weekly_survey') openSurvey();
      }

      _taps = FirebaseMessaging.onMessageOpenedApp.listen(onTap);
      onTap(await messaging.getInitialMessage());
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<void> _bind(Api api, String fid) async {
    final old = await _storage.read(key: 'weekly_fid');
    await api.request(
      '/weekly/device',
      body: {'installation_id': fid, 'enabled': true},
    );
    await _storage.write(key: 'weekly_fid', value: fid);
    if (old != null && old != fid)
      await api.request(
        '/weekly/device',
        body: {'installation_id': old, 'enabled': false},
      );
  }

  Future<void> enable(Api api, String owner) async {
    if (await _ready != true)
      throw StateError('push configuration unavailable');
    final messaging = FirebaseMessaging.instance;
    final permission = await messaging.requestPermission();
    if (permission.authorizationStatus != AuthorizationStatus.authorized)
      throw StateError('permission unavailable');
    final registration = messaging.onRegistered.first.timeout(
      const Duration(seconds: 12),
    );
    final values = await Future.wait<dynamic>([
      registration,
      messaging.register(),
    ]);
    if (_currentOwner?.call() != owner) throw StateError('session changed');
    await _bind(api, values.first as String);
    await _storage.write(key: 'weekly_owner', value: owner);
    enabledFor.value = owner;
  }

  Future<void> reconnect(Api api, String owner) async {
    if (await _ready != true || enabledFor.value != owner) return;
    await FirebaseMessaging.instance.register();
  }

  Future<void> disable(Api api) async {
    if (!supported || await _ready != true || enabledFor.value == null) return;
    final owner = enabledFor.value;
    enabledFor.value = null;
    try {
      await _queue;
      final fid = await _storage.read(key: 'weekly_fid');
      if (fid != null)
        await api.request(
          '/weekly/device',
          body: {'installation_id': fid, 'enabled': false},
        );
      await FirebaseMessaging.instance.unregister();
      await _storage.delete(key: 'weekly_fid');
      await _storage.delete(key: 'weekly_owner');
    } catch (_) {
      enabledFor.value = owner;
      rethrow;
    }
  }

  Future<void> signOut(Api api) async {
    if (!supported || enabledFor.value == null) return;
    try {
      await disable(api);
    } catch (_) {
      enabledFor.value = null;
      await _storage.delete(key: 'weekly_owner');
      try {
        await FirebaseMessaging.instance.unregister();
      } catch (_) {}
    }
  }

  void dispose() {
    _registrations?.cancel();
    _taps?.cancel();
    _currentOwner = null;
    _ready = null;
  }
}
