import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import '../auth/session.dart';

class JobSearch extends ConsumerStatefulWidget {
  const JobSearch({super.key, required this.open});
  final Future<void> Function(Uri) open;
  @override
  ConsumerState<JobSearch> createState() => _JobSearchState();
}

class _JobSearchState extends ConsumerState<JobSearch> {
  final skills = TextEditingController();
  String confirmed = '';
  List<dynamic>? items;
  bool busy = false, failed = false;
  @override
  void dispose() {
    skills.dispose();
    super.dispose();
  }

  Future<void> search() async {
    final text = skills.text.trim();
    if (text.isEmpty || busy) return;
    setState(() {
      confirmed = text;
      busy = true;
      items = null;
      failed = false;
    });
    try {
      final data = await ref
          .read(apiProvider)
          .request('/support/vacancies', body: {'text': text, 'area': '160'});
      if (mounted) setState(() => items = data['items'] as List);
    } catch (_) {
      if (mounted) setState(() => failed = true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  String matchReason(String title) {
    final matches =
        confirmed
            .split(RegExp(r'\s+'))
            .where(
              (word) =>
                  word.length > 1 &&
                  title.toLowerCase().contains(word.toLowerCase()),
            )
            .toSet();
    return matches.isEmpty
        ? 'В выдаче HH по запросу «$confirmed»'
        : 'В названии: ${matches.join(', ')}';
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Text('Работа в Алматы', style: Theme.of(context).textTheme.titleLarge),
      const SizedBox(height: 12),
      TextField(
        controller: skills,
        enabled: !busy,
        maxLength: 160,
        onChanged: (_) => setState(() {}),
        onSubmitted: (_) => search(),
        decoration: const InputDecoration(
          labelText: 'Навыки или профессия',
          hintText: 'Например, ремонт техники',
        ),
      ),
      FilledButton(
        onPressed: busy || skills.text.trim().isEmpty ? null : search,
        child: Text(busy ? 'Ищем вакансии…' : 'Искать на HeadHunter'),
      ),
      if (failed)
        const Text(
          'HeadHunter сейчас недоступен. Повторите поиск или откройте hh.kz.',
        ),
      if (items?.isEmpty == true)
        const Text(
          'По этим словам в Алматы вакансий не найдено. Попробуйте другую формулировку.',
        ),
      if (items?.isNotEmpty == true) ...[
        Text('HeadHunter подобрал вакансии по запросу «$confirmed» в Алматы.'),
        for (final v in items!)
          Card(
            child: ListTile(
              title: Text(v['title'] as String),
              subtitle: Text('\n'),
              trailing: const Icon(Icons.open_in_new),
              onTap: () => widget.open(Uri.parse(v['url'] as String)),
            ),
          ),
      ],
      if (confirmed.isNotEmpty)
        TextButton(
          onPressed:
              () => widget.open(
                Uri.https('hh.kz', '/search/vacancy', {
                  'area': '160',
                  'text': confirmed,
                }),
              ),
          child: const Text('Открыть этот поиск на hh.kz'),
        ),
    ],
  );
}
