import { useEffect } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { getTenantSlug } from '../../helpers';
import Loading from '../common/ui/Loading';

const DEFAULT_TENANT = 'lurus';

const ZitadelRedirect = ({ register = false }) => {
  const { tenantSlug: routeSlug } = useParams();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    // Priority: route param > query param > localStorage > default
    const slug =
      routeSlug ||
      searchParams.get('tenant') ||
      getTenantSlug() ||
      DEFAULT_TENANT;

    let url = `/api/v2/${slug}/auth/login?redirect_url=/oauth/zitadel`;
    if (register) {
      url += '&register=true';
    }
    window.location.href = url;
  }, [register, routeSlug, searchParams]);

  return <Loading />;
};

export default ZitadelRedirect;
