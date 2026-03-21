import { useState, useEffect, useCallback } from 'react';
import { API } from '../../helpers';

export default function useWalletData() {
  const [wallet, setWallet] = useState(null);
  const [loading, setLoading] = useState(true);

  const fetchWallet = useCallback(async () => {
    try {
      setLoading(true);
      const res = await API.get('/api/wallet/info', { skipErrorHandler: true });
      if (res.data.success) {
        setWallet(res.data.data);
      }
    } catch {
      // Platform unavailable — wallet will be null, UI degrades gracefully.
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWallet();
  }, [fetchWallet]);

  return { wallet, loading, refresh: fetchWallet };
}
