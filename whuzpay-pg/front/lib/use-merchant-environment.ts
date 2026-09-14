"use client";

import { useCallback, useEffect, useState } from "react";
import {
  getMerchantEnvironment,
  setMerchantEnvironment,
  type MerchantEnvironment,
} from "./merchant-auth";

/** Shared sandbox/production switch for merchant dashboard. */
export function useMerchantEnvironment() {
  const [environment, setEnv] = useState<MerchantEnvironment>("sandbox");

  useEffect(() => {
    // Reads localStorage, which doesn't exist during SSR — the state
    // initializer above must stay a plain "sandbox" default so server and
    // client markup match on first paint. Computing this during render
    // instead of here would cause a hydration mismatch whenever the stored
    // value differs from the default.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setEnv(getMerchantEnvironment());
    function onStorage(e: StorageEvent) {
      if (e.key === "merchant_environment") {
        setEnv(getMerchantEnvironment());
      }
    }
    function onCustom(e: Event) {
      const detail = (e as CustomEvent<MerchantEnvironment>).detail;
      if (detail === "sandbox" || detail === "production") {
        setEnv(detail);
      } else {
        setEnv(getMerchantEnvironment());
      }
    }
    window.addEventListener("storage", onStorage);
    window.addEventListener("merchant-environment-change", onCustom);
    return () => {
      window.removeEventListener("storage", onStorage);
      window.removeEventListener("merchant-environment-change", onCustom);
    };
  }, []);

  const setEnvironment = useCallback((env: MerchantEnvironment) => {
    setMerchantEnvironment(env);
    setEnv(env);
  }, []);

  return { environment, setEnvironment };
}
