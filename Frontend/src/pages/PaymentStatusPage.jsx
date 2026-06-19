import { useEffect, useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { verifyPaymentOrder } from '../api';
import { Spinner } from '../components/Spinner';

export default function PaymentStatusPage() {
  const [searchParams] = useSearchParams();
  const orderId = searchParams.get('order_id');
  const [status, setStatus] = useState('verifying');

  useEffect(() => {
    if (!orderId) {
      setStatus('failed');
      return;
    }

    const verify = async () => {
      try {
        const res = await verifyPaymentOrder({ order_id: orderId });
        if (res.data.status === 'paid_in_escrow') {
          setStatus('success');
        } else {
          setStatus('failed');
        }
      } catch (err) {
        console.error("Verification error", err);
        setStatus('failed');
      }
    };

    verify();
  }, [orderId]);

  return (
    <div style={{ minHeight: '60vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', textAlign: 'center', gap: '1rem' }}>
      {status === 'verifying' && (
        <>
          <Spinner />
          <h3>Verifying Payment...</h3>
          <p style={{ color: 'var(--text-secondary)' }}>Please do not close this window.</p>
        </>
      )}
      {status === 'success' && (
        <>
          <span style={{ fontSize: '4rem' }}>✅</span>
          <h2 style={{ color: 'var(--ink-muted)' }}>Payment Successful!</h2>
          <p style={{ color: 'var(--text-secondary)' }}>Your funds are held securely in Escrow. The seller has been notified to ship the item.</p>
          <Link to="/" className="btn btn--primary" style={{ marginTop: '1rem' }}>Return to Market</Link>
        </>
      )}
      {status === 'failed' && (
        <>
          <span style={{ fontSize: '4rem' }}>❌</span>
          <h2 style={{ color: 'var(--ink-muted)' }}>Payment Failed or Pending</h2>
          <p style={{ color: 'var(--text-secondary)' }}>We could not verify your payment at this time.</p>
          <Link to="/" className="btn btn--ghost" style={{ marginTop: '1rem' }}>← Back to market</Link>
        </>
      )}
    </div>
  );
}
