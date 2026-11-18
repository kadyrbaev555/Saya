import 'dart:async';
import 'dart:io';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:saya_app/core/network/api.dart';
import 'package:saya_app/core/security/vault.dart';
import 'package:saya_app/core/realtime/realtime.dart';
import 'package:saya_app/features/auth/session.dart';
import 'package:saya_app/features/assessments/assessment.dart';
import 'package:saya_app/features/surveys/survey.dart';

class MemoryVault extends Vault {
  String? token;
  @override
  Future<String?> read() async => token;
  @override
  Future<void> write(String? value) async {
    token = value;
  }
}

void main() {
  test(
    'Flutter receives RabbitMQ/Centrifugo publication and reloads REST profile',
    () async {
      final api = Api(secureVault: MemoryVault());
      addTearDown(api.close);
      final container = ProviderContainer(
        overrides: [apiProvider.overrideWithValue(api)],
      );
      addTearDown(container.dispose);
      await container
          .read(sessionProvider.notifier)
          .authenticate(
            'login',
            Platform.environment['SAYA_TEST_EMAIL']!,
            Platform.environment['SAYA_TEST_PASSWORD']!,
          );
      final state = container.read(sessionProvider);
      expect(state.hasError, isFalse);
      final connected = Completer<void>(), received = Completer<void>();
      final assessed = Completer<void>();
      final reconnected = Completer<void>();
      int connectionCount = 0;
      final listener = container.listen(realtimeProvider, (previous, next) {
        final data = next.asData?.value;
        if (data?['event'] == 'assessment.completed' && !assessed.isCompleted)
          assessed.complete();
        if (data?['status'] == 'connected') {
          connectionCount++;
          if (!connected.isCompleted) connected.complete();
          if (connectionCount >= 2 && !reconnected.isCompleted)
            reconnected.complete();
        }
        if (data?['event'] == 'demo.completed' && !received.isCompleted)
          received.complete();
      });
      addTearDown(listener.close);
      await connected.future.timeout(const Duration(seconds: 20));
      final id = state.asData!.value!['id'] as String;
      final process = await Process.start(
        'docker',
        [
          'compose',
          '--env-file',
          'backend/deployments/local/.env',
          '-f',
          'backend/deployments/local/compose.yaml',
          'run',
          '--rm',
          '--build',
          'demo',
        ],
        workingDirectory: '../..',
        environment: {...Platform.environment, 'DEMO_USER_ID': id},
      );
      unawaited(process.stdout.drain<void>());
      unawaited(process.stderr.drain<void>());
      await received.future.timeout(const Duration(seconds: 90));
      final profile = await container.read(profileProvider.future);
      expect(profile!['id'], id);
      expect(await process.exitCode, 0);
      final restart = await Process.run('docker', [
        'compose',
        '--env-file',
        'backend/deployments/local/.env',
        '-f',
        'backend/deployments/local/compose.yaml',
        'restart',
        'centrifugo',
      ], workingDirectory: '../..');
      expect(restart.exitCode, 0);
      await reconnected.future.timeout(const Duration(seconds: 30));
      expect((await api.request('/users/me'))['id'], id);
      final survey = await api.request('/surveys/current');
      final operation = await api.request(
        '/surveys/submissions',
        body: {
          'questionnaire_id': survey['id'],
          'version': survey['version'],
          'answers': {'immediate_threat': true, 'information_requested': true},
          'consent': true,
          'external_llm_consent': false,
          'idempotency_key': operationKey(),
        },
      );
      await assessed.future.timeout(const Duration(seconds: 30));
      final result = await container.read(
        assessmentProvider(operation['id'] as String).future,
      );
      expect(result['status'], 'completed');
      expect(result['provider'], 'demonstration-mock');
      expect(result['immediate_threat_reported'], isTrue);
      await api.logout();
    },
    skip: Platform.environment['SAYA_REALTIME_TEST'] != '1',
    timeout: const Timeout(Duration(minutes: 3)),
  );
}
