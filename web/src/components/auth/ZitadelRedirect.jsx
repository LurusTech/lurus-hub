import { useEffect } from 'react';
import Loading from '../common/ui/Loading';

const ZITADEL_LOGIN_URL =
  '/api/v2/lurus/auth/login?redirect_url=/oauth/zitadel';

const ZitadelRedirect = ({ register = false }) => {
  useEffect(() => {
    const url = register
      ? `${ZITADEL_LOGIN_URL}&register=true`
      : ZITADEL_LOGIN_URL;
    window.location.href = url;
  }, [register]);

  return <Loading />;
};

export default ZitadelRedirect;
