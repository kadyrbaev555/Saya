import 'package:material_ui/material_ui.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/network/api.dart';
import 'session.dart';

class RegistrationPage extends ConsumerStatefulWidget {
  const RegistrationPage({super.key});

  @override
  ConsumerState<RegistrationPage> createState() => _RegistrationPageState();
}

class _RegistrationPageState extends ConsumerState<RegistrationPage> {
  final form = GlobalKey<FormState>();
  final name = TextEditingController();
  final phone = TextEditingController();
  final email = TextEditingController();
  final password = TextEditingController();
  final code = TextEditingController();
  String? challenge, error;
  bool busy = false;

  @override
  void dispose() {
    for (final field in [name, phone, email, password, code]) {
      field.dispose();
    }
    super.dispose();
  }

  Future<void> submit({bool resend = false}) async {
    if (busy || (!resend && !form.currentState!.validate())) return;
    setState(() {
      busy = true;
      error = null;
    });
    try {
      if (challenge == null || resend) {
        final id = await ref
            .read(apiProvider)
            .startRegistration(
              name.text.trim(),
              phone.text.trim(),
              email.text.trim(),
              password.text,
            );
        if (mounted)
          setState(() {
            challenge = id;
            code.clear();
          });
      } else {
        await ref
            .read(sessionProvider.notifier)
            .confirmRegistration(challenge!, code.text);
        if (mounted) context.go('/profile');
      }
    } catch (e) {
      if (mounted)
        setState(() {
          error = switch (e) {
            ApiError(status: 401) =>
              'Код неверен или истёк. Проверьте письмо или запросите новый код.',
            ApiError(status: 429) =>
              'Повторное письмо можно запросить через минуту.',
            ApiError(status: 409) =>
              'Этот адрес уже зарегистрирован. Перейдите ко входу.',
            ApiError(status: 400) =>
              'Проверьте поля. Укажите телефон с кодом страны.',
            _ => 'Не удалось завершить запрос. Попробуйте ещё раз.',
          };
        });
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final verifying = challenge != null;
    return Scaffold(
      appBar: AppBar(
        title: Text(verifying ? 'Подтверждение почты' : 'Создать аккаунт'),
      ),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 440),
              child: Form(
                key: form,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (!verifying) ...[
                      TextFormField(
                        controller: name,
                        enabled: !busy,
                        autofillHints: const [AutofillHints.name],
                        decoration: const InputDecoration(labelText: 'ФИО'),
                        validator: requiredField,
                      ),
                      TextFormField(
                        controller: phone,
                        enabled: !busy,
                        keyboardType: TextInputType.phone,
                        decoration: const InputDecoration(
                          labelText: 'Телефон',
                          hintText: '+7 777 123 45 67',
                        ),
                        validator: requiredField,
                      ),
                      TextFormField(
                        controller: email,
                        enabled: !busy,
                        keyboardType: TextInputType.emailAddress,
                        decoration: const InputDecoration(labelText: 'Email'),
                        validator: requiredField,
                      ),
                      TextFormField(
                        controller: password,
                        enabled: !busy,
                        obscureText: true,
                        decoration: const InputDecoration(labelText: 'Пароль'),
                        validator:
                            (v) =>
                                (v?.length ?? 0) < 12
                                    ? 'Введите минимум 12 символов'
                                    : null,
                      ),
                    ] else ...[
                      Text(
                        'Отправили шестизначный код на ${email.text.trim()}. Он действует 10 минут.',
                      ),
                      TextFormField(
                        controller: code,
                        enabled: !busy,
                        autofocus: true,
                        keyboardType: TextInputType.number,
                        autofillHints: const [AutofillHints.oneTimeCode],
                        maxLength: 6,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                        decoration: const InputDecoration(
                          labelText: 'Код из письма',
                        ),
                        validator:
                            (v) => v?.length == 6 ? null : 'Введите 6 цифр',
                      ),
                    ],
                    if (error != null)
                      Semantics(liveRegion: true, child: Text(error!)),
                    const SizedBox(height: 24),
                    FilledButton(
                      onPressed: busy ? null : () => submit(),
                      child: Text(
                        verifying ? 'Подтвердить и войти' : 'Получить код',
                      ),
                    ),
                    if (verifying) ...[
                      TextButton(
                        onPressed: busy ? null : () => submit(resend: true),
                        child: const Text('Отправить код ещё раз'),
                      ),
                      TextButton(
                        onPressed:
                            busy
                                ? null
                                : () => setState(() {
                                  challenge = null;
                                  error = null;
                                  code.clear();
                                }),
                        child: const Text('Изменить данные'),
                      ),
                    ],
                    TextButton(
                      onPressed: busy ? null : () => context.go('/login'),
                      child: const Text('Уже есть аккаунт'),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  String? requiredField(String? value) =>
      (value ?? '').trim().isEmpty ? 'Заполните поле' : null;
}
