import { register } from "./register";
import { test, expect } from "@playwright/test";

test("регистрация, вход и восстановление браузерной сессии", async ({
  page,
  request,
}) => {
  const { email, password } = await register(page, request);
  await page.getByRole("button", { name: "Выйти" }).click();
  await page.setViewportSize({ width: 375, height: 800 });
  await page.getByLabel("Электронная почта").fill(email);
  await page.getByLabel("Пароль", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await expect(page).toHaveURL(/\/profile$/);
  await page.reload();
  await expect(page.getByText(email, { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Выйти" }).click();
  await page.goto("/profile");
  await expect(page).toHaveURL(/\/login$/);
});
