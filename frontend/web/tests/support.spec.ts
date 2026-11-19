import { test, expect } from "@playwright/test";
import { register } from "./register";

test("направления поддержки и более ранние опросы", async ({
  page,
  request,
  context,
}) => {
  await page.goto("/support/emergency");
  await expect(page.getByRole("link", { name: /112/ })).toHaveAttribute(
    "href",
    "tel:112",
  );
  await register(page, request);
  await page.setViewportSize({ width: 375, height: 800 });
  await page.getByRole("link", { name: "На главную" }).click();
  await page.getByRole("link", { name: /Жильё/ }).click();
  await context.grantPermissions(["geolocation"]);
  await context.setGeolocation({ latitude: 43.209, longitude: 76.898 });
  await page.getByRole("button", { name: "Подобрать район рядом" }).click();
  await expect(page.getByRole("link", { name: /Krisha/ })).toHaveAttribute(
    "href",
    /almaty-bostandykskij/,
  );
  await page.getByLabel("Район", { exact: true }).selectOption("3");
  await page.route("https://tile.openstreetmap.org/**", (route) =>
    route.abort(),
  );
  await page.getByRole("button", { name: "Показать мини-карту" }).click();
  await expect(page.getByRole("status")).toContainText("Карта не загрузилась");
  await expect(page.getByRole("link", { name: /Krisha/ })).toHaveAttribute(
    "href",
    /almaty-medeuskij/,
  );
  await page.getByRole("link", { name: /На главную/ }).click();
  await page.getByRole("link", { name: /Социальная поддержка/ }).click();
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page
    .getByLabel("Что вы хотите рассказать?")
    .fill("Демонстрационное обращение");
  await page.getByRole("button", { name: "Скопировать обращение" }).click();
  await expect(page.getByRole("status")).toHaveText("Обращение скопировано.");
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    "Демонстрационное обращение",
  );
  await page.getByRole("link", { name: /На главную/ }).click();
  await page.getByRole("link", { name: /Поиск работы/ }).click();
  const label = page.getByLabel("Навыки или профессия"),
    submit = page.getByRole("button", { name: /Искать/ });
  let mode = "success";
  await page.route("**/api/v1/support/vacancies", (route) => {
    expect(route.request().postDataJSON()).toEqual({
      text: "ремонт",
      area: "160",
    });
    return route.fulfill({
      status: mode === "failure" ? 503 : 200,
      json: {
        items:
          mode === "success"
            ? [
                {
                  title: "Мастер",
                  employer: "Мастерская",
                  url: "https://hh.kz/vacancy/1",
                },
              ]
            : [],
      },
    });
  });
  await label.fill("ремонт");
  await submit.click();
  await expect(
    page.getByRole("link", { name: "Мастер", exact: true }),
  ).toBeVisible();
  mode = "empty";
  await submit.click();
  await expect(page.getByRole("status")).toContainText("не найдено");
  mode = "failure";
  await submit.click();
  await expect(page.getByRole("alert")).toContainText("недоступен");
  await expect(page.getByRole("link", { name: /этот поиск/ })).toHaveAttribute(
    "href",
    "https://hh.kz/search/vacancy?" +
      new URLSearchParams({ area: "160", text: "ремонт" }),
  );
  await page.goto("/history");
  await expect(
    page.getByText("Здесь появятся результаты завершённых опросов."),
  ).toBeVisible();
  await page.route("**/api/v1/weekly/history*", (route) => {
    const older =
      new URL(route.request().url()).searchParams.get("before") === "older";
    return route.fulfill({
      json: {
        entries: [
          {
            id: older ? "a" : "b",
            completed_at: older ? 100 : 200,
            version: 2,
            percentage: older ? 0 : 20,
            ratings: { work: older ? 0 : 4 },
            directions: [],
          },
        ],
        next_cursor: older ? "" : "older",
        reminder_due: false,
        next_due: 0,
      },
    });
  });
  await page.reload();
  await page.getByRole("button", { name: "Более ранние опросы" }).click();
  await expect(page.getByText("Изменение · +20 п. п.")).toBeVisible();
  await expect(page.getByText("Самооценка · 0%")).toBeVisible();
});
