import React, { useContext, useEffect, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { UserContext } from '../context/User';
import { StatusContext } from '../context/Status';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Icon,
  Menu,
  Segment,
} from 'semantic-ui-react';
import {
  API,
  clearNacosEmbeddedConsoleLocalSession,
  getLogo,
  getSystemName,
  hasTenantPermission,
  isAdmin,
  isRoot,
  isMobile,
  isNacosEnabled,
  isTenantAdmin,
  isTenantConsoleDelegate,
  clearTenantConsoleActingTenantId,
  showSuccess,
} from '../helpers';
import Footer from './Footer';
import NacosThemeToggle from './NacosThemeToggle';
import TenantConsoleActingBar from './TenantConsoleActingBar';
import '../index.css';

const publicPaths = new Set([
  '/login',
  '/tenant-login',
  '/register',
  '/reset',
  '/user/reset',
]);

function isPublicLayoutPath(pathname) {
  if (publicPaths.has(pathname)) return true;
  if (pathname.startsWith('/oauth/')) return true;
  return false;
}

const nacosNavGroup = {
  type: 'group',
  groupId: 'nacos',
  name: 'header.nacos_menu',
  icon: 'cloud',
  admin: true,
  collapsible: true,
  children: [
    {
      name: 'header.nacos_sub_console',
      openNativeConsoleInNewTab: true,
      icon: 'external',
      admin: true,
    },
    {
      name: 'header.nacos_sub_namespaces',
      to: '/nacos/namespaces',
      icon: 'folder open',
      admin: true,
    },
    {
      name: 'header.nacos_sub_cs',
      to: '/nacos/cs',
      icon: 'settings',
      admin: true,
    },
    {
      name: 'header.nacos_sub_skills',
      to: '/nacos/skills',
      icon: 'code branch',
      admin: true,
    },
    {
      name: 'header.nacos_sub_agentspecs',
      to: '/nacos/agentspecs',
      icon: 'sitemap',
      admin: true,
    },
    {
      name: 'header.nacos_sub_mcp',
      to: '/nacos/mcp',
      icon: 'plug',
      admin: true,
    },
    {
      name: 'header.nacos_sub_a2a',
      to: '/nacos/a2a',
      icon: 'users',
      admin: true,
    },
    {
      name: 'header.nacos_sub_prompts',
      to: '/nacos/prompts',
      icon: 'file alternate outline',
      admin: true,
    },
    {
      name: 'header.nacos_sub_pipelines',
      to: '/nacos/pipelines',
      icon: 'tasks',
      admin: true,
    },
    {
      name: 'header.nacos_sub_perm',
      to: '/nacos/permissions',
      icon: 'shield alternate',
      admin: true,
    },
  ],
};

const tenantModuleGroup = {
  type: 'group',
  groupId: 'tenant',
  name: 'header.tenant_platform',
  icon: 'building',
  admin: true,
  collapsible: true,
  children: [
    { type: 'divider' },
    { type: 'header', name: 'header.tenant_platform_mgmt' },
    {
      name: 'header.tenant_upgrades',
      to: '/tenant-upgrades',
      icon: 'clipboard check',
    },
    {
      name: 'header.tenant_management',
      to: '/tenant-management',
      icon: 'building outline',
    },
    { type: 'divider' },
    { type: 'header', name: 'header.tenant_platform_console' },
    {
      name: 'header.tenant_subusers',
      to: '/tenant-console/users',
      icon: 'users',
    },
    {
      name: 'header.tenant_channels',
      to: '/tenant-console/channels',
      icon: 'sitemap',
    },
    {
      name: 'header.tenant_billing_reports',
      to: '/tenant-console/reports',
      icon: 'file text outline',
    },
  ],
};

const sidebarNavItems = [
  { name: 'header.home', to: '/', icon: 'home', admin: false, exact: true },
  nacosNavGroup,
  tenantModuleGroup,
  {
    name: 'header.tenant_reports',
    to: '/platform-reports',
    icon: 'file text',
    admin: true,
  },
  { name: 'header.routing', to: '/routing', icon: 'random', admin: true },
  {
    name: 'header.model_catalog',
    to: '/model-catalog',
    icon: 'list alternate outline',
    admin: true,
  },
  {
    name: 'header.operations_management',
    to: '/operations',
    icon: 'sliders horizontal',
    admin: true,
    rootOnly: true,
  },
  { name: 'header.channel', to: '/channel', icon: 'sitemap', admin: true },
  { name: 'header.token', to: '/token', icon: 'key', admin: false },
  { name: 'header.redemption', to: '/redemption', icon: 'dollar sign', admin: true },
  { name: 'header.topup', to: '/topup', icon: 'cart', admin: false },
  { name: 'header.user', to: '/user', icon: 'user', admin: true },
  { name: 'header.dashboard', to: '/dashboard', icon: 'chart bar', admin: false },
  { name: 'header.log', to: '/log', icon: 'book', admin: false },
  { name: 'header.setting', to: '/setting', icon: 'setting', admin: false },
  { name: 'header.about', to: '/about', icon: 'info circle', admin: false },
];

if (typeof localStorage !== 'undefined' && localStorage.getItem('chat_link')) {
  sidebarNavItems.splice(1, 0, {
    name: 'header.chat',
    to: '/chat',
    icon: 'comments',
    admin: false,
  });
}

function isNavActive(locPath, item) {
  if (!item.to) return false;
  if (item.exact) {
    return locPath === item.to;
  }
  if (locPath === item.to) {
    return true;
  }
  return locPath.startsWith(`${item.to}/`);
}

function isNavGroupChildActive(locPath, group) {
  if (!group.children) return false;
  return group.children.some((ch) => !ch.openNativeConsoleInNewTab && isNavActive(locPath, ch));
}

function navGroupModifierClass(groupId) {
  if (groupId === 'nacos') return ' app-sidebar-nav-group--nacos';
  if (groupId === 'tenant') return ' app-sidebar-nav-group--tenant';
  return '';
}

function userDisplayName(user) {
  if (!user) return '';
  const d = user.display_name && String(user.display_name).trim();
  return d || user.username || '';
}

function userInitial(user) {
  const n = userDisplayName(user);
  return n ? n.charAt(0).toUpperCase() : '?';
}

const SidebarLayout = ({ children }) => {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    try {
      return localStorage.getItem('one-api-sidebar-collapsed') === '1';
    } catch {
      return false;
    }
  });
  const nacosMenuEnabled =
    statusState.status?.nacos_enabled !== false &&
    statusState.status?.nacos_enabled !== 'false' &&
    isNacosEnabled();
  const [nacosSectionOpen, setNacosSectionOpen] = useState(() => {
    try {
      return localStorage.getItem('one-api-sidebar-nacos-open') === '1';
    } catch {
      return false;
    }
  });

  const [tenantSectionOpen, setTenantSectionOpen] = useState(() => {
    try {
      return localStorage.getItem('one-api-sidebar-tenant-open') === '1';
    } catch {
      return false;
    }
  });

  useEffect(() => {
    if (location.pathname.startsWith('/nacos')) {
      setNacosSectionOpen(true);
    }
    if (
      location.pathname.startsWith('/tenant-upgrades') ||
      location.pathname.startsWith('/tenant-management') ||
      location.pathname.startsWith('/tenant-console')
    ) {
      setTenantSectionOpen(true);
    }
  }, [location.pathname]);

  const systemName = getSystemName();
  const logo = getLogo();

  const isEnglishUI =
    i18n.language && String(i18n.language).toLowerCase().startsWith('en');

  const toggleLanguage = async () => {
    await i18n.changeLanguage(isEnglishUI ? 'zh' : 'en');
    window.location.reload();
  };

  const toggleSidebarCollapsed = () => {
    setSidebarCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('one-api-sidebar-collapsed', next ? '1' : '0');
      } catch {
        /* ignore */
      }
      return next;
    });
  };

  async function logout() {
    setMobileOpen(false);
    await API.get('/api/user/logout');
    showSuccess('注销成功!');
    userDispatch({ type: 'logout' });
    localStorage.removeItem('user');
    clearTenantConsoleActingTenantId();
    clearNacosEmbeddedConsoleLocalSession();
    navigate('/login');
  }

  const renderTopBarAccount = (compact) => (
    <div
      className={
        compact ? 'app-topbar-actions app-topbar-actions--mobile' : 'app-topbar-actions'
      }
    >
      <Button
        type='button'
        basic
        icon
        size={compact ? 'mini' : 'small'}
        className='app-theme-toggle'
        onClick={toggleLanguage}
        title={t('header.language_switch_tooltip')}
        aria-label={t('header.language_switch_tooltip')}
      >
        <Icon name='language' />
      </Button>
      <NacosThemeToggle compact={compact} />
      {userState.user ? (
        <>
          <div
            className={
              compact ? 'app-topbar-user app-topbar-user--compact' : 'app-topbar-user'
            }
            title={userDisplayName(userState.user)}
          >
            <div className='app-topbar-avatar'>
              {userInitial(userState.user)}
            </div>
            {!compact && (
              <span className='app-topbar-name'>
                {userDisplayName(userState.user)}
              </span>
            )}
          </div>
          <Button basic size='small' onClick={logout}>
            {t('header.logout')}
          </Button>
        </>
      ) : (
        <Button.Group size='small'>
          <Button onClick={() => navigate('/login')}>{t('header.login')}</Button>
          <Button onClick={() => navigate('/register')}>{t('header.register')}</Button>
        </Button.Group>
      )}
    </div>
  );

  const mainPageShell = (body) => (
    <div className='app-main-page-wrap'>
      <div className='app-main-page-panel'>
        <div className='app-main-page-scroll'>
          {location.pathname.startsWith('/tenant-console') &&
          isAdmin() &&
          !isTenantAdmin() ? (
            <TenantConsoleActingBar />
          ) : null}
          {body}
        </div>
        <div className='app-main-page-footer'>
          <Footer />
        </div>
      </div>
    </div>
  );

  if (isPublicLayoutPath(location.pathname)) {
    return (
      <div className='app-sidebar-layout-fill'>
        <div className='app-public-wrap'>
          <div className='app-public-scroll'>{children}</div>
          <div className='app-public-footer'>
            <Footer />
          </div>
        </div>
      </div>
    );
  }

  const renderNav = (onPick, navCollapsed) => {
    if (isTenantAdmin() || isTenantConsoleDelegate()) {
      const tenantItems = [
        {
          name: 'header.tenant_console',
          to: '/tenant-console',
          icon: 'building',
          exact: true,
        },
      ];
      if (
        hasTenantPermission('manage_users') ||
        hasTenantPermission('manage_tokens')
      ) {
        tenantItems.push({
          name: 'header.tenant_subusers',
          to: '/tenant-console/users',
          icon: 'users',
        });
      }
      if (hasTenantPermission('manage_channels')) {
        tenantItems.push({
          name: 'header.tenant_channels',
          to: '/tenant-console/channels',
          icon: 'sitemap',
        });
      }
      if (hasTenantPermission('manage_billing')) {
        tenantItems.push({
          name: 'header.tenant_billing_reports',
          to: '/tenant-console/reports',
          icon: 'file text outline',
        });
      }
      if (isTenantAdmin()) {
        tenantItems.push(
          { name: 'header.token', to: '/token', icon: 'key' },
          { name: 'header.log', to: '/log', icon: 'book' },
          { name: 'header.dashboard', to: '/dashboard', icon: 'chart bar' },
          { name: 'header.topup', to: '/topup', icon: 'cart' },
        );
      }
      tenantItems.push(
        { name: 'header.setting', to: '/setting', icon: 'setting' },
        { name: 'header.about', to: '/about', icon: 'info circle' },
      );

      const showTenantNacos =
        nacosMenuEnabled &&
        (isTenantAdmin() || hasTenantPermission('manage_nacos'));

      const flatItems = tenantItems.map((item) => {
        const active = isNavActive(location.pathname, item);
        return (
          <Menu.Item
            key={item.to + item.name}
            active={active}
            title={navCollapsed ? t(item.name) : undefined}
            onClick={() => {
              navigate(item.to);
              if (onPick) onPick();
            }}
          >
            <Icon name={item.icon} />
            <span className='app-sidebar-nav-label'>{t(item.name)}</span>
          </Menu.Item>
        );
      });

      const groupActive = isNavGroupChildActive(location.pathname, nacosNavGroup);
      const collapsible = !!nacosNavGroup.collapsible;
      const showNacosChildren =
        !collapsible || nacosSectionOpen || navCollapsed;
      const toggleNacos = () => {
        if (!collapsible || navCollapsed) return;
        setNacosSectionOpen((open) => {
          const next = !open;
          try {
            localStorage.setItem('one-api-sidebar-nacos-open', next ? '1' : '0');
          } catch {
            /* ignore */
          }
          return next;
        });
      };

      const nacosBlock = showTenantNacos ? (
        <Menu.Item
          key='tenant-nacos-group'
          className={`app-sidebar-nav-group app-sidebar-nav-group--nacos${
            groupActive ? ' app-sidebar-nav-group--active' : ''
          }`}
          title={navCollapsed ? t(nacosNavGroup.name) : undefined}
        >
          <Menu.Header
            className={`app-sidebar-nav-group-header${
              collapsible ? ' app-sidebar-nav-group-header--collapsible' : ''
            }`}
            onClick={collapsible && !navCollapsed ? toggleNacos : undefined}
          >
            {collapsible && !navCollapsed ? (
              <Icon
                name={nacosSectionOpen ? 'angle down' : 'angle right'}
                className='app-sidebar-nav-group-chevron'
              />
            ) : null}
            <Icon name={nacosNavGroup.icon} />
            <span className='app-sidebar-nav-label'>{t(nacosNavGroup.name)}</span>
          </Menu.Header>
          {showNacosChildren ? (
            <Menu.Menu>
              {nacosNavGroup.children.map((ch) => {
                const subActive = ch.openNativeConsoleInNewTab
                  ? false
                  : isNavActive(location.pathname, ch);
                return (
                  <Menu.Item
                    key={ch.openNativeConsoleInNewTab ? ch.name : ch.to}
                    active={subActive}
                    title={navCollapsed ? t(ch.name) : undefined}
                    onClick={() => {
                      if (ch.openNativeConsoleInNewTab) {
                        window.open(
                          `${window.location.origin}/nacos-ui/`,
                          '_blank',
                          'noopener,noreferrer',
                        );
                        if (onPick) onPick();
                        return;
                      }
                      navigate(ch.to);
                      if (onPick) onPick();
                    }}
                  >
                    <Icon name={ch.icon} />
                    <span className='app-sidebar-nav-label'>{t(ch.name)}</span>
                  </Menu.Item>
                );
              })}
            </Menu.Menu>
          ) : null}
        </Menu.Item>
      ) : null;

      return (
        <>
          {nacosBlock}
          {flatItems}
        </>
      );
    }
    return (
      <>
        {sidebarNavItems.map((item) => {
      if (item.type === 'group') {
        if (item.groupId === 'nacos' && !nacosMenuEnabled) return null;
        if (item.admin && !(isAdmin() || isTenantAdmin())) return null;
        const groupActive = isNavGroupChildActive(location.pathname, item);
        const collapsible = !!item.collapsible;
        const isNacos = item.groupId === 'nacos';
        const isTenant = item.groupId === 'tenant';
        const isOpen = isNacos ? nacosSectionOpen : isTenant ? tenantSectionOpen : false;
        const showChildren = !collapsible || isOpen || navCollapsed;

        const toggleGroup = () => {
          if (!collapsible || navCollapsed) return;
          if (isNacos) {
            setNacosSectionOpen((open) => {
              const next = !open;
              try { localStorage.setItem('one-api-sidebar-nacos-open', next ? '1' : '0'); } catch {}
              return next;
            });
          } else if (isTenant) {
            setTenantSectionOpen((open) => {
              const next = !open;
              try { localStorage.setItem('one-api-sidebar-tenant-open', next ? '1' : '0'); } catch {}
              return next;
            });
          }
        };

        return (
          <Menu.Item
            key={item.groupId || item.name}
            className={`app-sidebar-nav-group${navGroupModifierClass(item.groupId)}${
              groupActive ? ' app-sidebar-nav-group--active' : ''
            }`}
            title={navCollapsed ? t(item.name) : undefined}
          >
            <Menu.Header
              className={`app-sidebar-nav-group-header${
                collapsible ? ' app-sidebar-nav-group-header--collapsible' : ''
              }`}
              onClick={collapsible && !navCollapsed ? toggleGroup : undefined}
            >
              {collapsible && !navCollapsed ? (
                <Icon
                  name={isOpen ? 'angle down' : 'angle right'}
                  className='app-sidebar-nav-group-chevron'
                />
              ) : null}
              <Icon name={item.icon} />
              <span className='app-sidebar-nav-label'>{t(item.name)}</span>
            </Menu.Header>
            {showChildren ? (
              <Menu.Menu>
                {item.children.map((ch, idx) => {
                  if (ch.type === 'divider') {
                    return (
                      <div
                        key={`divider-${idx}`}
                        className='app-sidebar-nav-section-divider'
                        role='separator'
                        aria-hidden='true'
                      />
                    );
                  }
                  if (ch.type === 'header') {
                    return (
                      <Menu.Header
                        key={`header-${idx}`}
                        className='app-sidebar-nav-subheader'
                      >
                        {t(ch.name)}
                      </Menu.Header>
                    );
                  }
                  const subActive = ch.openNativeConsoleInNewTab
                    ? false
                    : isNavActive(location.pathname, ch);
                  return (
                    <Menu.Item
                      key={ch.openNativeConsoleInNewTab ? ch.name : ch.to}
                      active={subActive}
                      title={navCollapsed ? t(ch.name) : undefined}
                      onClick={() => {
                        if (ch.openNativeConsoleInNewTab) {
                          window.open(
                            `${window.location.origin}/nacos-ui/`,
                            '_blank',
                            'noopener,noreferrer',
                          );
                          if (onPick) onPick();
                          return;
                        }
                        navigate(ch.to);
                        if (onPick) onPick();
                      }}
                    >
                      <Icon name={ch.icon} />
                      <span className='app-sidebar-nav-label'>{t(ch.name)}</span>
                    </Menu.Item>
                  );
                })}
              </Menu.Menu>
            ) : null}
          </Menu.Item>
        );
      }
      if (item.admin && !isAdmin()) return null;
      if (item.rootOnly && !isRoot()) return null;
      const active = isNavActive(location.pathname, item);
      return (
        <Menu.Item
          key={item.to + item.name}
          active={active}
          title={navCollapsed ? t(item.name) : undefined}
          onClick={() => {
            navigate(item.to);
            if (onPick) onPick();
          }}
        >
          <Icon name={item.icon} />
          <span className='app-sidebar-nav-label'>{t(item.name)}</span>
        </Menu.Item>
      );
    })}
      </>
    );
  };

  if (isMobile()) {
    return (
      <div className='app-sidebar-layout-fill'>
        <div className='app-shell-mobile'>
        <Segment basic className='app-mobile-top'>
          <Link
            to='/'
            onClick={() => setMobileOpen(false)}
            style={{
              display: 'flex',
              alignItems: 'center',
              minWidth: 0,
              flex: 1,
              marginRight: 8,
              textDecoration: 'none',
              overflow: 'hidden',
            }}
          >
            <img src={logo} alt='logo' style={{ height: 28, marginRight: 8, flexShrink: 0 }} />
            <b
              style={{
                color: 'var(--app-chrome-text)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {systemName}
            </b>
          </Link>
          <div className='app-mobile-top-right'>
            {renderTopBarAccount(true)}
            <Icon
              name={mobileOpen ? 'close' : 'sidebar'}
              size='large'
              link
              onClick={() => setMobileOpen(!mobileOpen)}
            />
          </div>
        </Segment>
        {mobileOpen ? (
          <Menu
            vertical
            fluid
            className='app-sidebar-menu app-sidebar-menu--mobile'
            style={{
              margin: 0,
              borderRadius: 0,
            }}
          >
            {renderNav(() => setMobileOpen(false), false)}
          </Menu>
        ) : null}
        <div className='app-mobile-main'>
          {mainPageShell(children)}
        </div>
      </div>
      </div>
    );
  }

  return (
    <div className='app-sidebar-layout-fill'>
      <div
        className={
          sidebarCollapsed
            ? 'app-shell app-shell--sidebar-collapsed'
            : 'app-shell'
        }
      >
        <aside
          className={
            sidebarCollapsed ? 'app-sidebar app-sidebar--collapsed' : 'app-sidebar'
          }
        >
        <div className='app-sidebar-brand'>
          <Link to='/' className='app-sidebar-brand-link'>
            <img src={logo} alt='logo' className='app-sidebar-logo' />
            <span className='app-sidebar-title'>{systemName}</span>
          </Link>
        </div>
        <Menu
          vertical
          fluid
          className='app-sidebar-menu'
          style={{
            border: 'none',
            boxShadow: 'none',
            borderRadius: 0,
            margin: 0,
          }}
        >
          {renderNav(null, sidebarCollapsed)}
        </Menu>
        <div className='app-sidebar-footer'>
          <Button
            type='button'
            basic
            fluid
            icon
            className='app-sidebar-toggle'
            onClick={toggleSidebarCollapsed}
            title={
              sidebarCollapsed
                ? t('header.sidebar_expand')
                : t('header.sidebar_collapse')
            }
            aria-label={
              sidebarCollapsed
                ? t('header.sidebar_expand')
                : t('header.sidebar_collapse')
            }
          >
            <Icon
              name={sidebarCollapsed ? 'angle double right' : 'angle double left'}
            />
          </Button>
        </div>
      </aside>
      <div className='app-main'>
        <header className='app-topbar'>
          <div className='app-topbar-leading' aria-hidden='true' />
          {renderTopBarAccount(false)}
        </header>
        <div className='app-main-content'>{mainPageShell(children)}</div>
      </div>
    </div>
    </div>
  );
};

export default SidebarLayout;
