import 'dart:async';
import 'dart:convert';
import 'package:centrifuge/centrifuge.dart' as centrifuge;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../features/auth/session.dart';

final profileProvider = FutureProvider<Map<String, dynamic>?>((ref) async {
  if (ref.watch(sessionProvider).asData?.value == null) return null;
  return ref.read(apiProvider).request('/users/me');
});
final realtimeProvider = StreamProvider.autoDispose<Map<String, dynamic>>((
  ref,
) {
  final user = ref.watch(sessionProvider).asData?.value;
  if (user == null) return const Stream.empty();
  final api = ref.read(apiProvider);
  final channel = 'user:${user['id']}:assessments';
  final events = StreamController<Map<String, dynamic>>();
  bool disposed = false;
  final subscriptions = <StreamSubscription>[];
  final client = centrifuge.createClient(
    const String.fromEnvironment(
      'WS_URL',
      defaultValue: 'ws://localhost:8000/connection/websocket',
    ),
    centrifuge.ClientConfig(
      getToken:
          (_) async =>
              (await api.request(
                    '/realtime/connection-token',
                    body: {},
                  ))['token']
                  as String,
    ),
  );
  final subscription = client.newSubscription(
    channel,
    centrifuge.SubscriptionConfig(
      getToken:
          (_) async =>
              (await api.request(
                    '/realtime/subscription-token',
                    body: {'channel': channel},
                  ))['token']
                  as String,
    ),
  );
  void emit(Map<String, dynamic> event) {
    if (!disposed) events.add(event);
  }

  subscriptions.add(
    client.connected.listen((_) {
      if (!disposed) ref.invalidate(profileProvider);
      emit({'status': 'connected'});
    }),
  );
  subscriptions.add(
    client.disconnected.listen((_) {
      emit({'status': 'disconnected'});
    }),
  );
  subscriptions.add(
    client.error.listen((_) {
      emit({'status': 'reconnecting'});
    }),
  );
  subscriptions.add(
    subscription.publication.listen((event) {
      try {
        final data =
            jsonDecode(utf8.decode(event.data)) as Map<String, dynamic>;
        if (data['version'] == 1) {
          if (!disposed) ref.invalidate(profileProvider);
          emit(data);
        }
      } catch (_) {
        emit({'status': 'invalid_event'});
      }
    }),
  );
  Future<void> start() async {
    try {
      await subscription.subscribe();
      if (!disposed) await client.connect();
    } catch (_) {
      emit({'status': 'reconnecting'});
    }
  }

  unawaited(start());
  ref.onDispose(() {
    disposed = true;
    for (final listener in subscriptions) {
      unawaited(listener.cancel());
    }
    unawaited(client.disconnect());
    unawaited(events.close());
  });
  return events.stream;
});
