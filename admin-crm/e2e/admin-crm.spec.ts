import { createHmac } from "node:crypto";
import { expect, test } from "@playwright/test";
import { promises as fs } from "node:fs";
import path from "node:path";

const outbox = path.resolve(__dirname, "../../backend/.admin-e2e-data/email_outbox");
const proof = path.resolve(__dirname, "../../docs/proof/sprint-4-admin-crm");
const backend = "http://127.0.0.1:18180";

function base32Decode(value: string) {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const character of value.replace(/=+$/, "").toUpperCase()) {
    bits += alphabet.indexOf(character).toString(2).padStart(5, "0");
  }
  const bytes: number[] = [];
  for (let index = 0; index+8 <= bits.length; index += 8) bytes.push(Number.parseInt(bits.slice(index, index+8), 2));
  return Buffer.from(bytes);
}

function totp(secret: string) {
  const counter = Math.floor(Date.now() / 30_000);
  const value = Buffer.alloc(8);
  value.writeBigUInt64BE(BigInt(counter));
  const digest = createHmac("sha1", base32Decode(secret)).update(value).digest();
  const offset = digest[digest.length - 1] & 0x0f;
  const code = (digest.readUInt32BE(offset) & 0x7fffffff) % 1_000_000;
  return code.toString().padStart(6, "0");
}

async function verificationToken(recipient: string) {
  const deadline = Date.now() + 20_000;
  while (Date.now() < deadline) {
    const entries = await fs.readdir(outbox).catch(() => [] as string[]);
    for (const name of entries) {
      const content = await fs.readFile(path.join(outbox, name), "utf8");
      if (!content.includes(`To: ${recipient}`) || !content.includes("/auth/verify-email")) continue;
      const match = content.match(/[?&]token=([^&\s<"]+)/);
      if (match) return decodeURIComponent(match[1]);
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`No verification token arrived for ${recipient}`);
}

async function capture(page: import("@playwright/test").Page, name: string, project: string) {
  await fs.mkdir(proof, { recursive: true });
  await page.screenshot({ path: path.join(proof, `${name}-${project}.png`), fullPage: true });
}

test("administrator completes secure access and operates every CRM module", async ({ page, request }, testInfo) => {
  const project = testInfo.project.name;
  const email = `admin-${project}@example.com`;
  const password = "AdminOperationsPassword!42";

  const registration = await request.post(`${backend}/api/v1/auth/register`, {
    headers: { Origin: "http://127.0.0.1:13100" },
    data: { email, password, country: "ZA", dateOfBirth: "1990-01-01", acceptTerms: true, acceptFairPlay: true }
  });
  expect(registration.status()).toBe(201);
  const token = await verificationToken(email);
  const verification = await request.post(`${backend}/api/v1/auth/verify-email`, {
    headers: { Origin: "http://127.0.0.1:13100" },
    data: { token }
  });
  expect(verification.ok()).toBeTruthy();

  await page.goto("/login");
  await capture(page, "login", project);
  await page.getByLabel("Work email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Continue securely" }).click();
  await expect(page).toHaveURL(/\/mfa\/setup$/);
  await expect(page.getByRole("heading", { name: "Add the secret" })).toBeVisible();
  const secretDisplay = page.locator(".secret-row code");
  await expect(secretDisplay).not.toHaveText("Generating secure secret...");
  const secret = await secretDisplay.textContent();
  expect(secret).toBeTruthy();
  await page.getByLabel("Six-digit code").fill(totp(secret || ""));
  const confirm = page.getByRole("button", { name: "Confirm authenticator" });
  await expect(confirm).toBeEnabled();
  await confirm.click({ force: true });
  await expect(page.getByRole("heading", { name: "Store recovery codes" })).toBeVisible();
  await capture(page, "mfa-enrollment", project);
  await page.getByRole("button", { name: "Enter operations" }).click();

  const pages = [
    ["/", "Command center", "dashboard"],
    ["/users", "Player records", "users"],
    ["/finance", "Money movement", "finance"],
    ["/compliance", "Identity, risk, and jurisdiction", "compliance"],
    ["/support", "Player support", "support"],
    ["/audit", "Audit center", "audit"],
    ["/announcements", "Notices and announcements", "announcements"],
    ["/monitoring", "System health", "monitoring"]
  ];
  for (const [url, heading, screenshot] of pages) {
    await page.goto(url);
    await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible();
    await expect(page.locator(".spin")).toHaveCount(0);
    await capture(page, screenshot, project);
  }

  if (project.startsWith("mobile")) {
    await page.getByRole("button", { name: "Open navigation" }).click();
  }
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
});
