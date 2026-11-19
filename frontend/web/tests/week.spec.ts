import { test, expect } from "@playwright/test";
import { register } from "./register";

test("опрос без внешнего ИИ, восстановление результата и ошибка обработки", async ({
  page,
  request,
}) => {
  await register(page, request);
  let focusCalls = 0;
  page.on("request", (req) => {
    if (req.url().endsWith("/interview/focus")) focusCalls++;
  });
  await page.getByRole("link", { name: /Пройти опрос/ }).click();
  await page.getByLabel("Ваш ответ").fill("Демонстрационный ответ");
  await page.getByRole("button", { name: "0", exact: true }).click();
  await page.getByRole("button", { name: "Дальше" }).click();
  for (let i = 0; i < 5; i++)
    await page.getByRole("button", { name: "Пропустить вопрос" }).click();
  await expect(
    page.getByRole("button", { name: "Сохранить результат" }),
  ).toBeDisabled();
  await page.getByLabel(/Подтверждаю описание/).check();
  const submitted = page.waitForRequest((req) =>
    req.url().endsWith("/surveys/submissions"),
  );
  await page.getByRole("button", { name: "Сохранить результат" }).click();
  const payload = (await submitted).postDataJSON();
  expect(payload.external_llm_consent).toBe(false);
  expect(payload.ratings).toEqual({ housing: 0 });
  expect(focusCalls).toBe(0);
  await expect(
    page.getByRole("heading", { name: "Результат готов" }),
  ).toBeVisible({ timeout: 30000 });
  await page.reload();
  await expect(page.getByText("Жильё · 0/4")).toBeVisible();
  await expect(page.getByText(/Общий процент появится/)).toBeVisible();

  // Hold REST in pending state so a fast local worker cannot finish before subscription.
  const id = page.url().split("/").pop();
  let status = "pending";
  await page.route("**/api/v1/assessments/" + id, (route) =>
    route.fulfill({ json: { status, version: 2 } }),
  );
  let publish: (() => void) | undefined;
  await page.routeWebSocket("**/connection/websocket", (socket) => {
    const server = socket.connectToServer();
    let channel = "";
    socket.onMessage((message) => {
      for (const line of String(message).split("\n").filter(Boolean)) {
        const command = JSON.parse(line);
        if (command.subscribe) channel = command.subscribe.channel;
      }
      server.send(message);
    });
    server.onMessage((message) => {
      socket.send(message);
      for (const line of String(message).split("\n").filter(Boolean)) {
        if (!JSON.parse(line).subscribe) continue;
        publish = () =>
          socket.send(
            JSON.stringify({
              push: {
                channel,
                pub: {
                  data: {
                    event: "assessment.failed",
                    version: 1,
                    assessment_id: id,
                    status: "failed",
                  },
                },
              },
            }),
          );
      }
    });
  });
  await page.reload();
  await expect.poll(() => Boolean(publish)).toBe(true);
  await expect(page.getByText("Подготовка результата…")).toBeVisible();
  status = "failed";
  publish!();
  publish!();
  await expect(
    page.getByRole("heading", { name: "Анализ недоступен" }),
  ).toBeVisible({ timeout: 1500 });
});
