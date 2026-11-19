import { test, expect } from "@playwright/test";

for (const width of [320, 1440]) {
  test(`Клавиатурная навигация при ширине ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/");
    const email = page.getByLabel("Электронная почта");
    await expect(email).toBeVisible();
    await page.keyboard.press("Tab");
    await expect(email).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(page.getByLabel("Пароль", { exact: true })).toBeFocused();
    const emergency = page.getByRole("link", {
      name: "Экстренные номера Казахстана",
    });
    for (
      let i = 0;
      i < 6 &&
      !(await emergency.evaluate((el) => el === document.activeElement));
      i++
    )
      await page.keyboard.press("Tab");
    await expect(emergency).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Экстренная помощь" }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.reload();
    await expect(page.getByRole("link", { name: /112/ })).toHaveAttribute(
      "href",
      "tel:112",
    );
  });
}
