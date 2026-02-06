const TENANT_SLUG_KEY = 'tenant_slug';
const DEFAULT_TENANT = 'lurus';

export function isV2Mode() {
  return !!localStorage.getItem(TENANT_SLUG_KEY);
}

export function getTenantSlug() {
  return localStorage.getItem(TENANT_SLUG_KEY) || '';
}

export function setTenantSlug(slug) {
  if (slug) {
    localStorage.setItem(TENANT_SLUG_KEY, slug);
  }
}

export function clearTenantSlug() {
  localStorage.removeItem(TENANT_SLUG_KEY);
}

// Build V2 API path: /api/v2/{slug}{path}
export function v2Url(path) {
  const slug = getTenantSlug() || DEFAULT_TENANT;
  return `/api/v2/${slug}${path}`;
}
