export const usingFakeMinecraftServer = (): boolean => {
  return Deno.env.get("USING_MCSERVER_FAKE") === "yes";
};
