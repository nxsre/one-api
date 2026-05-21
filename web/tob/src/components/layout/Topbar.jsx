import { useLocation } from 'react-router-dom';
import { ROUTE_TITLES } from '@/config/routes';

export default function Topbar() {
  const { pathname } = useLocation();
  const title = ROUTE_TITLES[pathname] || '控制台';

  return (
    <header className="tob-topbar">
      <span className="tob-topbar-title">{title}</span>
      <div style={{ flex: 1 }} />
    </header>
  );
}
