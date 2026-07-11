import { NavLink, useNavigate } from 'react-router-dom';
import { NAV_SECTIONS } from '@/config/routes';
import LogoMark from '@/components/brand/LogoMark';
import { NavIcon } from '@/components/icons/NavIcons';
import { useUser } from '@/context/UserContext';
import { logout as apiLogout } from '@/lib/auth';
import { getUserDisplayName, getUserSurname } from '@/lib/userDisplay';

function LogoutIcon() {
  return (
    <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" strokeLinecap="round" />
      <polyline points="16 17 21 12 16 7" strokeLinecap="round" strokeLinejoin="round" />
      <line x1="21" y1="12" x2="9" y2="12" strokeLinecap="round" />
    </svg>
  );
}

export default function Sidebar() {
  const { user, logout: clearUser } = useUser();
  const navigate = useNavigate();
  const surname = getUserSurname(user);
  const displayName = getUserDisplayName(user);

  const handleLogout = async () => {
    await apiLogout();
    clearUser();
    navigate('/login', { replace: true });
  };

  return (
    <aside className="tob-sidebar">
      <div className="tob-sidebar-logo">
        <div className="tob-logo-icon">
          <LogoMark size={18} fill="#fff" />
        </div>
        <span className="tob-logo-text">TokenHub</span>
        <span className="tob-logo-badge">Pro</span>
      </div>

      <nav className="tob-nav">
        {NAV_SECTIONS.map((section, index) => (
          <div
            key={section.label || `section-${index}`}
            className={`tob-nav-section${section.label ? ' tob-nav-section--group' : ''}`}
          >
            {section.label ? <div className="tob-nav-label">{section.label}</div> : null}
            {section.items.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) => `tob-nav-item${isActive ? ' active' : ''}`}
              >
                <NavIcon name={item.icon} />
                {item.label}
              </NavLink>
            ))}
          </div>
        ))}
      </nav>

      <div className="tob-sidebar-footer">
        <button type="button" className="tob-user-card" onClick={handleLogout} title="退出登录">
          <div className="tob-avatar" aria-hidden>
            {surname}
          </div>
          <div className="tob-user-meta">
            <span className="tob-user-name">{displayName}</span>
            <span className="tob-user-logout">
              <LogoutIcon />
              退出登录
            </span>
          </div>
        </button>
      </div>
    </aside>
  );
}
