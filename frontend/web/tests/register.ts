import { randomUUID } from "node:crypto";
import { expect, type Page, type APIRequestContext } from "@playwright/test";

export async function register(page: Page, request: APIRequestContext) {
  const email = "web-" + randomUUID() + "@example.test";
  const password = randomUUID();
  const mailbox = process.env.SAYA_TEST_MAILPIT_URL || "http://127.0.0.1:8025";
  await page.goto("/profile");
  await expect(page).toHaveURL(/\/login$/);
  await page.getByRole("link", { name: "Зарегистрироваться" }).click();
  await page.getByLabel("Полное имя").fill("Тест Saya");
  await page.getByLabel("Телефон").fill("+77000000001");
  await page.getByLabel("Электронная почта").fill(email);
  await page.getByLabel("Пароль", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Создать аккаунт" }).click();
  await expect(page.getByLabel("Код из письма")).toBeVisible();
  let code = "";
  await expect
    .poll(async () => {
      const search = await request.get(mailbox + "/api/v1/search", {
        params: { query: "to:" + email },
      });
      const first = (await search.json()).messages?.[0];
      if (first) {
        const mail = await request.get(mailbox + "/api/v1/message/" + first.ID);
        code = (await mail.json()).Text.match(/\b\d{6}\b/)?.[0] || "";
      }
      return Boolean(code);
    })
    .toBe(true);
  await page.getByLabel("Код из письма").fill(code);
  await page.getByRole("button", { name: "Подтвердить почту" }).click();
  await expect(page.getByText(email, { exact: true })).toBeVisible();
  return { email, password };
}
