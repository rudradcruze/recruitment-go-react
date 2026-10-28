export interface DemoAdmin {
  emailAddress: string;
  password: string;
}

// demoAdmin gives the demo Admin of the settings VITE_DEMO_ADMIN_EMAIL and VITE_DEMO_ADMIN_PASSWORD.
// It gives nothing when a setting is empty, so a real deployment leaves them empty and shows no account.
// Vite puts the values in the files of the page, so they are public. Use a demo account only.
export function demoAdmin(): DemoAdmin | undefined {
  const emailAddress: unknown = import.meta.env.VITE_DEMO_ADMIN_EMAIL;
  const password: unknown = import.meta.env.VITE_DEMO_ADMIN_PASSWORD;
  if (
    typeof emailAddress === "string" &&
    emailAddress !== "" &&
    typeof password === "string" &&
    password !== ""
  ) {
    return { emailAddress, password };
  }
  return undefined;
}
