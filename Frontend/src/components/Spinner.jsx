import './Spinner.css';

export function Spinner({ size = 'md', className = '' }) {
  return (
    <div className={`spinner spinner--${size} ${className}`} aria-label="Loading" />
  );
}

export default function PageLoader() {
  return (
    <div className="page-loader">
      <div className="page-loader__inner">
       
        <Spinner size="lg" />
      </div>
    </div>
  );
}
