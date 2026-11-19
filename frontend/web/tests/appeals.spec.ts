import { test, expect } from "@playwright/test";
import { register } from "./register";

test("владелец открывает привязку карты у провайдера", async ({ page }) => {
  const user = { id: "test-owner", email: "owner@example.test", role: "USER" };
  await page.route("**/api/v1/auth/csrf", (r) =>
    r.fulfill({ json: { csrf_token: "fixture" } }),
  );
  await page.route("**/api/v1/auth/refresh", (r) =>
    r.fulfill({ json: { access_token: "fixture", user } }),
  );
  await page.route("**/api/v1/users/me", (r) => r.fulfill({ json: user }));
  await page.route("**/api/v1/appeals/mine", (r) =>
    r.fulfill({
      json: {
        checkout_available: true,
        items: [
          {
            id: "fixture",
            state: "pending",
            description: "Описание",
            card_ready: false,
          },
        ],
      },
    }),
  );
  await page.route("**/api/v1/appeals/fixture/payout-card", (r) =>
    r.fulfill({ json: { url: "https://sandbox.example.test/card" } }),
  );
  await page.route("https://sandbox.example.test/card", (r) =>
    r.fulfill({ body: "Тестовая привязка" }),
  );
  await page.goto("/appeals/mine");
  await page.getByRole("button", { name: /Привязать тестовую карту/ }).click();
  await expect(page).toHaveURL("https://sandbox.example.test/card");
});

test("гость переходит к тестовой оплате только для доступной анкеты", async ({
  page,
}) => {
  let amount = 0;
  await page.route("**/api/v1/appeals", (route) =>
    route.fulfill({
      json: {
        items: [
          { id: "fixture", description: "Доступная анкета", card_ready: true },
        ],
        checkout_available: true,
      },
    }),
  );
  await page.route("**/api/v1/appeals/fixture/checkout", (route) => {
    amount = route.request().postDataJSON().amount;
    return route.fulfill({
      json: { url: "https://sandbox.example.test/checkout" },
    });
  });
  await page.route("https://sandbox.example.test/checkout", (route) =>
    route.fulfill({ body: "Тестовая страница провайдера" }),
  );
  await page.goto("/appeals");
  await page.getByRole("button", { name: "1000 ₸" }).click();
  await expect(page).toHaveURL("https://sandbox.example.test/checkout");
  expect(amount).toBe(1000);
});

test("подача, модерация, жалоба и снятие анкеты", async ({
  page,
  request,
  browser,
}) => {
  const adminEmail = process.env.SAYA_TEST_ADMIN_EMAIL,
    adminPassword = process.env.SAYA_TEST_ADMIN_PASSWORD;
  test.skip(!adminEmail || !adminPassword, "Нужен тестовый модератор");
  await register(page, request);
  await page.goto("/appeals/review");
  await expect(page).toHaveURL(/\/profile$/);
  await page.getByRole("link", { name: /Пройти опрос/ }).click();
  await page.getByRole("button", { name: "Пропустить вопрос" }).click();
  await page.getByRole("button", { name: "1", exact: true }).click();
  await page.getByRole("button", { name: "Дальше" }).click();
  for (let i = 0; i < 4; i++)
    await page.getByRole("button", { name: "Пропустить вопрос" }).click();
  await page.getByLabel("Денежная поддержка", { exact: true }).check();
  await page.getByLabel(/Подтверждаю описание/).check();
  await page.getByRole("button", { name: "Сохранить результат" }).click();
  await page.getByRole("button", { name: /подготовить анкету/ }).click();
  const payload = "Тестовая анкета " + crypto.randomUUID();
  const send = page.getByRole("button", { name: "Отправить на проверку" });
  await page.getByLabel("Публичное описание").fill("Черновик");
  await page.getByLabel(/Согласен опубликовать/).check();
  await page.getByLabel("Публичное описание").fill(payload);
  await expect(send).toBeDisabled();
  await page.getByLabel(/Согласен опубликовать/).check();
  await send.click();
  await expect(page.getByText("На модерации", { exact: true })).toBeVisible();
  const guest = await browser.newPage();
  const admin = await browser.newPage();
  try {
    const catalogLoaded = guest.waitForResponse(
      (r) => new URL(r.url()).pathname === "/api/v1/appeals" && r.ok(),
    );
    await guest.goto("/appeals");
    expect(
      (await (await catalogLoaded).json()).items.some(
        (a: { description: string }) => a.description === payload,
      ),
    ).toBe(false);
    await expect(
      guest.getByRole("heading", { name: "Помочь человеку" }),
    ).toBeVisible();
    await expect(guest.getByText(payload)).toHaveCount(0);
    await admin.goto("/login");
    await admin.getByLabel("Электронная почта").fill(adminEmail!);
    await admin.getByLabel("Пароль", { exact: true }).fill(adminPassword!);
    await admin.getByRole("button", { name: "Войти", exact: true }).click();
    await admin.getByRole("link", { name: "Профиль", exact: true }).click();
    await admin.getByRole("link", { name: "Модерация анкет" }).click();
    const review = admin.getByRole("article").filter({ hasText: payload });
    await review.getByRole("button", { name: "Одобрить" }).click();
    await expect(
      admin.getByRole("status").filter({ hasText: "Решение сохранено" }),
    ).toBeVisible();
    await guest.getByRole("button", { name: "Обновить список" }).click();
    const published = guest.getByRole("article").filter({ hasText: payload });
    await expect(
      published.getByText(/платёж для этой анкеты пока недоступен/),
    ).toBeVisible();
    await published.getByRole("button", { name: "Пожаловаться" }).click();
    await expect(
      guest.getByRole("status").filter({ hasText: "Жалоба" }),
    ).toBeVisible();
    await admin.getByRole("button", { name: "Обновить список" }).click();
    await expect(review.getByText("Поступила жалоба")).toBeVisible();
    await review.getByRole("button", { name: "Отклонить" }).click();
    await expect(review).toHaveCount(0);
    await page.getByRole("button", { name: "Обновить список" }).click();
    await expect(page.getByText("Отклонена")).toBeVisible();
    await page.getByRole("button", { name: "Снять анкету" }).click();
    await expect(page.getByText("Снята", { exact: true })).toBeVisible();
    const refreshed = guest.waitForResponse(
      (r) => new URL(r.url()).pathname === "/api/v1/appeals" && r.ok(),
    );
    await guest.getByRole("button", { name: "Обновить список" }).click();
    expect(
      (await (await refreshed).json()).items.some(
        (a: { description: string }) => a.description === payload,
      ),
    ).toBe(false);
    await expect(published).toHaveCount(0);
  } finally {
    await guest.close();
    await admin.close();
  }
});
