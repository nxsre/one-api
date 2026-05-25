import { NavLink, useNavigate } from 'react-router-dom';
import { NAV_SECTIONS } from '@/config/routes';
import LogoMark from '@/components/brand/LogoMark';
import { NavIcon } from '@/components/icons/NavIcons';
import { useUser } from '@/context/UserContext';
import { logout as apiLogout } from '@/lib/auth';

export default function Sidebar() {
  const { user, logout: clearUser } = useUser();
  const navigate = useNavigate();
  const initial = (user?.username || user?.name || 'U').charAt(0).toUpperCase();

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
          <div className="tob-avatar">{initial}</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="tob-user-name">{user?.display_name || user?.username || '用户'}</div>
            <div className="tob-user-plan">点击退出</div>
          </div>
        </button>
      </div>
    </aside>
  );
}
