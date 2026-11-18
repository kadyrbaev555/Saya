import 'dart:async';
import 'weekly.dart';
import '../support/appeals.dart';
import 'package:go_router/go_router.dart';
import '../surveys/survey.dart';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/realtime/realtime.dart';
import '../auth/session.dart';

final latestSurveyProvider = FutureProvider.autoDispose<String?>((ref) async {
  final owner = ref.watch(sessionProvider).asData?.value?['id'];
  if (owner == null) return null;
  final data = await ref.watch(apiProvider).request('/surveys/latest');
  return data['id'] as String?;
});

final assessmentProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, id) async {
      ref.watch(sessionProvider.select((s) => s.asData?.value?['id']));
      ref.watch(realtimeProvider);
      final result = await ref.read(apiProvider).request('/assessments/$id');
      if (result['status'] == 'pending') {
        final timer = Timer(
          const Duration(seconds: 2),
          () => ref.invalidateSelf(),
        );
        ref.onDispose(timer.cancel);
      }
      return result;
    });

class AssessmentPage extends ConsumerWidget {
  final String id;
  const AssessmentPage({super.key, required this.id});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final result = ref.watch(assessmentProvider(id));
    return Scaffold(
      appBar: AppBar(title: const Text('Результат')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 640),
          child: result.when(
            loading: () => const CircularProgressIndicator(),
            error:
                (e, s) => Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text('Не удалось получить результат'),
                    TextButton(
                      onPressed: () => ref.invalidate(assessmentProvider(id)),
                      child: const Text('Повторить'),
                    ),
                  ],
                ),
            data:
                (data) => ListView(
                  padding: const EdgeInsets.all(24),
                  children: [
                    Text(
                      data['status'] == 'pending'
                          ? 'Подготовка результата'
                          : data['status'] == 'failed'
                          ? 'Анализ недоступен'
                          : 'Результат готов',
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                    const SizedBox(height: 20),
                    Text(data['disclaimer'] as String? ?? ''),
                    if (data['version'] == 2 &&
                        data['status'] == 'completed') ...[
                      const WeeklyPushButton(),
                      if (data['percentage'] != null)
                        Text(
                          'Моя оценка недели — ${data['percentage']}%',
                          style: Theme.of(context).textTheme.titleLarge,
                        ),
                      Text(
                        data['percentage'] == null
                            ? 'Общий процент появится при заполнении всех пяти оценок.'
                            : 'Сумма ваших пяти оценок из 20 возможных, переведённая в проценты.',
                      ),
                      for (final e in ((data['ratings'] as Map?) ?? {}).entries)
                        Text('${topicLabels[e.key]} · ${e.value}/4'),
                      if ((data['summary'] ?? '') != '')
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          child: Text(data['summary']),
                        ),
                      for (final key in (data['directions'] as List? ?? []))
                        if (key == 'financial')
                          Builder(
                            builder: (_) {
                              final entries =
                                  ref
                                          .watch(weeklyProvider(''))
                                          .asData
                                          ?.value['entries']
                                      as List? ??
                                  [];
                              if (entries.isEmpty || entries.first['id'] != id)
                                return const SizedBox.shrink();
                              return OutlinedButton(
                                onPressed:
                                    () => editAppeal(
                                      context,
                                      ref,
                                      id,
                                      data['summary'] as String? ?? '',
                                    ),
                                child: const Text(
                                  'Денежная поддержка · подготовить анкету',
                                ),
                              );
                            },
                          )
                        else if (needLabels.containsKey(key))
                          OutlinedButton(
                            onPressed: () => context.push('/support/$key'),
                            child: Text(needLabels[key]!),
                          ),
                    ],
                    TextButton(
                      onPressed: () => context.go('/profile'),
                      child: const Text('Вернуться в профиль'),
                    ),
                    if (data['provider'] == 'demonstration-mock')
                      const Text('Использован демонстрационный Mock LLM'),
                    if (data['status'] == 'pending')
                      const LinearProgressIndicator(),
                    for (final recommendation
                        in (data['recommendations'] as List? ?? []))
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 12),
                        child: Text(recommendation as String),
                      ),
                  ],
                ),
          ),
        ),
      ),
    );
  }
}
