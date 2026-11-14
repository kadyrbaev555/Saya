import 'dart:math';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../auth/session.dart';

final surveyProvider = FutureProvider(
  (ref) => ref.read(apiProvider).request('/surveys/current'),
);
String operationKey() {
  final random = Random.secure();
  final b = List<int>.generate(16, (_) => random.nextInt(256));
  b[6] = (b[6] & 15) | 64;
  b[8] = (b[8] & 63) | 128;
  final s = b.map((v) => v.toRadixString(16).padLeft(2, '0')).join();
  return '${s.substring(0, 8)}-${s.substring(8, 12)}-${s.substring(12, 16)}-${s.substring(16, 20)}-${s.substring(20)}';
}

class SurveyPage extends ConsumerStatefulWidget {
  const SurveyPage({super.key});
  @override
  ConsumerState<SurveyPage> createState() => _SurveyPageState();
}

class _SurveyPageState extends ConsumerState<SurveyPage> {
  final answers = <String, bool>{
    'immediate_threat': false,
    'information_requested': false,
  };
  final key = operationKey();
  bool consent = false, external = false, loading = false;
  String? error;
  @override
  Widget build(BuildContext context) {
    final survey = ref.watch(surveyProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('Небольшой опрос')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 640),
          child: survey.when(
            loading: () => const CircularProgressIndicator(),
            error:
                (e, s) => Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text('Опрос недоступен'),
                    TextButton(
                      onPressed: () => ref.invalidate(surveyProvider),
                      child: const Text('Повторить'),
                    ),
                  ],
                ),
            data:
                (data) => ListView(
                  padding: const EdgeInsets.all(24),
                  children: [
                    Text(data['disclaimer'] as String),
                    for (final q in data['questions'] as List)
                      SwitchListTile(
                        title: Text(q['label'] as String),
                        value: answers[q['id']] ?? false,
                        onChanged:
                            loading
                                ? null
                                : (v) => setState(
                                  () => answers[q['id'] as String] = v,
                                ),
                      ),
                    CheckboxListTile(
                      title: const Text(
                        'Добровольно согласен на обработку демонстрационных ответов',
                      ),
                      value: consent,
                      onChanged:
                          loading
                              ? null
                              : (v) => setState(() => consent = v ?? false),
                    ),
                    CheckboxListTile(
                      title: const Text(
                        'Отдельно разрешаю передачу демонстрационных ответов внешнему LLM, если он настроен. Условия обработки должны быть проверены оператором.',
                      ),
                      value: external,
                      onChanged:
                          loading
                              ? null
                              : (v) => setState(() => external = v ?? false),
                    ),
                    if (error != null) Text(error!),
                    FilledButton(
                      onPressed:
                          !consent || loading
                              ? null
                              : () async {
                                setState(() {
                                  loading = true;
                                  error = null;
                                });
                                try {
                                  final result = await ref
                                      .read(apiProvider)
                                      .request(
                                        '/surveys/submissions',
                                        body: {
                                          'questionnaire_id': data['id'],
                                          'version': data['version'],
                                          'answers': answers,
                                          'consent': consent,
                                          'external_llm_consent': external,
                                          'idempotency_key': key,
                                        },
                                      );
                                  if (context.mounted)
                                    context.go('/assessment/${result['id']}');
                                } catch (e) {
                                  if (mounted)
                                    setState(
                                      () =>
                                          error =
                                              'Не удалось отправить. Можно повторить запрос.',
                                    );
                                } finally {
                                  if (mounted) setState(() => loading = false);
                                }
                              },
                      child:
                          loading
                              ? const CircularProgressIndicator()
                              : const Text('Отправить демонстрационные ответы'),
                    ),
                  ],
                ),
          ),
        ),
      ),
    );
  }
}
