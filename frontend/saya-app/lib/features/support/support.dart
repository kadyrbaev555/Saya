import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../auth/session.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:url_launcher/url_launcher.dart';

enum Direction {
  housing('housing', 'Жильё', 'Посуточная аренда', Icons.home_outlined),
  jobs('jobs', 'Поиск работы', 'Вакансии по вашим навыкам', Icons.work_outline),
  psychologist(
    'psychologist',
    'Психолог',
    'Подготовить обращение',
    Icons.forum_outlined,
  ),
  social(
    'social',
    'Социальная поддержка',
    'Волонтёрские организации',
    Icons.people_outline,
  ),
  emergency(
    'emergency',
    'Экстренная помощь',
    'Номера служб Казахстана',
    Icons.phone_outlined,
  );

  const Direction(this.path, this.title, this.description, this.icon);
  final String path, title, description;
  final IconData icon;
}

class SupportHome extends StatelessWidget {
  const SupportHome({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('SAYA'),
      actions: [
        TextButton.icon(
          onPressed: () => context.push('/profile'),
          icon: const Icon(Icons.person_outline),
          label: const Text('Профиль'),
        ),
      ],
    ),
    body: SafeArea(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 800),
          child: ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Какая помощь вам нужна?',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 16),
              for (final item in Direction.values)
                Card(
                  child: ListTile(
                    leading: Icon(item.icon),
                    title: Text(item.title),
                    subtitle: Text(item.description),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => context.push('/support/${item.path}'),
                  ),
                ),
            ],
          ),
        ),
      ),
    ),
  );
}

class PublicAppeals extends StatelessWidget {
  const PublicAppeals({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Помочь человеку')),
    body: const Center(
      child: Padding(
        padding: EdgeInsets.all(24),
        child: Text('Пока нет опубликованных анкет для помощи.'),
      ),
    ),
  );
}

class DirectionPage extends ConsumerStatefulWidget {
  const DirectionPage(this.direction, {super.key});
  final Direction direction;
  @override
  ConsumerState<DirectionPage> createState() => _DirectionPageState();
}

class _DirectionPageState extends ConsumerState<DirectionPage> {
  final draft = TextEditingController();
  bool edited = false;
  bool filled = false;
  @override
  void initState() {
    super.initState();
    if (widget.direction == Direction.psychologist ||
        widget.direction == Direction.social)
      loadSummary();
  }

  Future<void> loadSummary() async {
    try {
      final api = ref.read(apiProvider);
      final id = (await api.request('/surveys/latest'))['id'] as String?;
      if (!mounted || id == null || id.isEmpty) return;
      final result = await api.request('/assessments/$id');
      final summary = result['summary'] as String?;
      if (mounted &&
          !edited &&
          result['status'] == 'completed' &&
          summary != null &&
          summary.isNotEmpty) {
        setState(() {
          draft.text = summary;
          filled = true;
        });
      }
    } catch (_) {
      /* Manual entry remains available when the result cannot load. */
    }
  }

  @override
  void dispose() {
    draft.dispose();
    super.dispose();
  }

  Future<void> open(Uri uri) async {
    try {
      if (await launchUrl(uri)) return;
    } on PlatformException {
      // Some devices have no application registered for telephone links.
    }
    if (mounted)
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Не удалось открыть ${uri.toString()}')),
      );
  }

  Widget link(String label, Uri Function() uri) => FilledButton.tonalIcon(
    onPressed: () => open(uri()),
    icon: const Icon(Icons.open_in_new),
    label: Text(label),
  );

  @override
  Widget build(BuildContext context) {
    final direction = widget.direction;
    return Scaffold(
      appBar: AppBar(title: Text(direction.title)),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 680),
          child: ListView(
            padding: const EdgeInsets.all(24),
            children: [
              if (direction == Direction.housing) ...[
                const Text('Выберите район и даты на карте посуточной аренды.'),
                const SizedBox(height: 16),
                link(
                  'Открыть карту Krisha.kz',
                  () => Uri.parse(
                    'https://m.krisha.kz/map/arenda/kvartiry-posutochno/',
                  ),
                ),
              ],
              if (direction == Direction.jobs) ...[
                TextField(
                  controller: draft,
                  decoration: const InputDecoration(
                    labelText: 'Навыки или профессия',
                    hintText: 'Например, ремонт техники',
                  ),
                ),
                const SizedBox(height: 16),
                link(
                  'Найти вакансии на hh.kz',
                  () => Uri.https('hh.kz', '/search/vacancy', {
                    'area': '40',
                    'text': draft.text.trim(),
                  }),
                ),
              ],
              if (direction == Direction.social ||
                  direction == Direction.psychologist) ...[
                TextField(
                  controller: draft,
                  onChanged: (_) => edited = true,
                  maxLines: 4,
                  decoration: const InputDecoration(
                    labelText: 'Что вы хотите рассказать?',
                    alignLabelWithHint: true,
                  ),
                ),
                const SizedBox(height: 8),
                TextButton.icon(
                  icon: const Icon(Icons.copy),
                  label: const Text('Скопировать обращение'),
                  onPressed: () async {
                    final message = draft.text.trim();
                    if (message.isEmpty) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('Введите текст обращения.'),
                        ),
                      );
                      return;
                    }
                    await Clipboard.setData(ClipboardData(text: message));
                    if (context.mounted)
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Обращение скопировано.')),
                      );
                  },
                ),
                const SizedBox(height: 16),
                if (direction == Direction.social)
                  link(
                    'Найти организацию на Qazvolunteer',
                    () => Uri.parse('https://qazvolunteer.kz/ru'),
                  ),
                if (direction == Direction.psychologist) ...[
                  const Text(
                    'Линия 150 оказывает бесплатную психологическую помощь детям, молодёжи и их семьям.',
                  ),
                  OutlinedButton.icon(
                    onPressed: () => open(Uri.parse('tel:150')),
                    icon: const Icon(Icons.phone_outlined),
                    label: const Text('Набрать 150'),
                  ),
                  const Divider(),
                  Text(
                    filled
                        ? 'Вы можете изменить описание из последнего опроса перед обращением.'
                        : 'В профиле можно пройти первоначальный опрос и получить рекомендации.',
                  ),
                  TextButton(
                    onPressed: () => context.push('/survey'),
                    child: const Text('Перейти к опросу'),
                  ),
                ],
              ],
              if (direction == Direction.emergency)
                for (final (number, label) in const [
                  ('112', 'Единая экстренная служба'),
                  ('101', 'Пожарные и спасатели'),
                  ('102', 'Полиция'),
                  ('103', 'Скорая медицинская помощь'),
                  ('104', 'Аварийная газовая служба'),
                ])
                  Card(
                    child: ListTile(
                      title: Text(number),
                      subtitle: Text(label),
                      trailing: const Icon(Icons.phone_outlined),
                      onTap: () => open(Uri.parse('tel:$number')),
                    ),
                  ),
            ],
          ),
        ),
      ),
    );
  }
}
