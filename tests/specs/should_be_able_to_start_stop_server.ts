const TARGET_HOST = Deno.env.get("TARGET_HOST");

const login = async (): Promise<string> => {
  const resp = await fetch(`${TARGET_HOST}/login`, {
    method: "POST",
    headers: {
      "Origin": TARGET_HOST,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ userName: "user1", password: "password1" }),
  });

  const { success } = await resp.json();
  if (!success) {
    console.error("Login error");
    Deno.exit(1);
  }

  return resp.headers.get("Set-Cookie");
};

console.log("Login");
const cookies = await login();

{
  console.log("Launch server");
  const params = new URLSearchParams();
  params.set("machine-type", "2g");
  params.set("server-version", "1.20.4");
  params.set("prefer-detect", "true");
  params.set("world-source", "new-world");
  params.set("world-name", `test-${Math.random()}`);
  params.set("seed", "");
  params.set("level-type", "default");

  const { success } = await fetch(`${TARGET_HOST}/api/launch`, {
    method: "POST",
    headers: {
      "Origin": TARGET_HOST,
      "Content-Type": "application/x-www-form-urlencoded",
      "Cookie": cookies,
    },
    body: params.toString(),
  }).then((resp) => resp.json());
  if (!success) {
    console.error("Launch failed");
    Deno.exit(1);
  }
}

console.log("Wait server launched");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await fetch(`${TARGET_HOST}/api/streaming/events`, {
    headers: {
      "Origin": TARGET_HOST,
      "Accept": "application/ld+json",
      "Cookie": cookies,
    },
  }).then((resp) => resp.json());

  if (pageCode === 3) {
    // control page
    break;
  } else if (pageCode === 4) {
    // manual setup page
    console.error("Unexpected page");
    Deno.exit(1);
  }
}

{
  console.log("Stop server");
  const { success } = await fetch(`${TARGET_HOST}/api/stop`, {
    method: "POST",
    headers: {
      "Origin": TARGET_HOST,
      "Cookie": cookies,
    },
  }).then((resp) => resp.json());
  if (!success) {
    console.error("Stop failed");
    Deno.exit(1);
  }
}

console.log("Wait server stopped");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await fetch(`${TARGET_HOST}/api/streaming/events`, {
    headers: {
      "Origin": TARGET_HOST,
      "Accept": "application/ld+json",
      "Cookie": cookies,
    },
  }).then((resp) => resp.json());

  if (pageCode === 1) {
    // control page
    break;
  } else if (pageCode === 4) {
    // manual setup page
    console.error("Unexpected page");
    Deno.exit(1);
  }
}

console.log("Success");
