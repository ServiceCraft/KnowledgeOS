export function channelModules(enabledModules?: Record<string, unknown>) {
  const raw = enabledModules?.channels;
  if (typeof raw === 'boolean') {
    return { telegram: raw, max: raw, vk: raw };
  }
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    const data = raw as Record<string, unknown>;
    return {
      telegram: data.telegram === true,
      max: data.max === true,
      vk: data.vk === true,
    };
  }
  return { telegram: false, max: false, vk: false };
}

export function handoffModule(enabledModules?: Record<string, unknown>) {
  return enabledModules?.handoff === true;
}

export function handoffFallbackText(enabledModules?: Record<string, unknown>) {
  const value = enabledModules?.handoff_fallback_text;
  return typeof value === 'string' ? value : '';
}
