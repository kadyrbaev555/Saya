import 'dart:async';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/realtime/realtime.dart';
import '../auth/session.dart';

final assessmentProvider = FutureProvider.autoDispose
    .family<Map<String, dynamic>, String>((ref, id) async {
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
