import React, { useContext, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { UserContext } from '../context/User';
import { useTranslation } from 'react-i18next';

import {
  Button,
  Container,
  Dropdown,
  Icon,
  Menu,
  Segment,
} from 'semantic-ui-react';
import {
  API,
  clearNacosEmbeddedConsoleLocalSession,
  getLogo,
  getSystemName,
  isAdmin,
  isMobile,
  showSuccess,
} from '../helpers';
import '../index.css';

// Header Buttons
let headerButtons = [
  {
    name: 'header.channel',
    to: '/channel',
    icon: 'sitemap',
    admin: true,
  },
  {
    name: 'header.routing',
    to: '/routing',
    icon: 'random',
    admin: true,
  },
  {
    name: 'header.nacos_skills',
    to: '/nacos/skills',
    icon: 'code branch',
    admin: true,
  },
  {
    name: 'header.nacos_perm',
    to: '/nacos/permissions',
    icon: 'shield alternate',
    admin: true,
  },
  {
    name: 'header.token',
    to: '/token',
    icon: 'key',
  },
  {
    name: 'header.redemption',
    to: '/redemption',
    icon: 'dollar sign',
    admin: true,
  },
  {
    name: 'header.topup',
    to: '/topup',
    icon: 'cart',
  },
  {
    name: 'header.user',
    to: '/user',
    icon: 'user',
    admin: true,
  },
  {
    name: 'header.dashboard',
    to: '/dashboard',
    icon: 'chart bar',
  },
  {
    name: 'header.log',
    to: '/log',
    icon: 'book',
  },
  {
    name: 'header.setting',
    to: '/setting',
    icon: 'setting',
  },
  {
    name: 'header.about',
    to: '/about',
    icon: 'info circle',
  },
];

if (localStorage.getItem('chat_link')) {
  headerButtons.splice(1, 0, {
    name: 'header.chat',
    to: '/chat',
    icon: 'comments',
  });
}

const Header = () => {
  const { t, i18n } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  let navigate = useNavigate();

  const [showSidebar, setShowSidebar] = useState(false);
  const systemName = getSystemName();
  const logo = getLogo();

  async function logout() {
    setShowSidebar(false);
    await API.get('/api/user/logout');
    showSuccess('注销成功!');
    userDispatch({ type: 'logout' });
    localStorage.removeItem('user');
    clearNacosEmbeddedConsoleLocalSession();
    navigate('/login');
  }

  const toggleSidebar = () => {
    setShowSidebar(!showSidebar);
  };

  const renderButtons = (isMobile) => {
    return headerButtons.map((button) => {
      if (button.admin && !isAdmin()) return <></>;
      if (isMobile) {
        return (
          <Menu.Item
            key={button.name}
            onClick={() => {
              navigate(button.to);
              setShowSidebar(false);
            }}
            style={{ fontSize: '15px' }}
          >
            {t(button.name)}
          </Menu.Item>
        );
      }
      return (
        <Menu.Item
          key={button.name}
          as={Link}
          to={button.to}
          style={{
            fontSize: '15px',
            fontWeight: '400',
            color: '#666',
          }}
        >
          <Icon name={button.icon} style={{ marginRight: '4px' }} />
          {t(button.name)}
        </Menu.Item>
      );
    });
  };

  const isEnglishUI =
    i18n.language && String(i18n.language).toLowerCase().startsWith('en');

  const toggleLanguage = async () => {
    await i18n.changeLanguage(isEnglishUI ? 'zh' : 'en');
    window.location.reload();
  };

  if (isMobile()) {
    return (
      <>
        <Menu
          borderless
          size='large'
          style={
            showSidebar
              ? {
                  borderBottom: 'none',
                  marginBottom: '0',
                  borderTop: 'none',
                  height: '51px',
                }
              : { borderTop: 'none', height: '52px' }
          }
        >
          <Container
            style={{
              width: '100%',
              maxWidth: isMobile() ? '100%' : '1200px',
              padding: isMobile() ? '0 10px' : '0 20px',
            }}
          >
            <Menu.Item as={Link} to='/'>
              <img src={logo} alt='logo' style={{ marginRight: '0.75em' }} />
              <div style={{ fontSize: '20px' }}>
                <b>{systemName}</b>
              </div>
            </Menu.Item>
            <Menu.Menu position='right'>
              <Menu.Item onClick={toggleSidebar}>
                <Icon name={showSidebar ? 'close' : 'sidebar'} />
              </Menu.Item>
            </Menu.Menu>
          </Container>
        </Menu>
        {showSidebar ? (
          <Segment style={{ marginTop: 0, borderTop: '0' }}>
            <Menu secondary vertical style={{ width: '100%', margin: 0 }}>
              {renderButtons(true)}
              <Menu.Item
                onClick={() => {
                  toggleLanguage();
                  setShowSidebar(false);
                }}
                style={{ fontSize: '15px' }}
              >
                <Icon name='language' style={{ marginRight: '8px' }} />
                {t('header.language_switch_tooltip')}
              </Menu.Item>
              <Menu.Item>
                {userState.user ? (
                  <Button onClick={logout} style={{ color: '#666666' }}>
                    {t('header.logout')}
                  </Button>
                ) : (
                  <>
                    <Button
                      onClick={() => {
                        setShowSidebar(false);
                        navigate('/login');
                      }}
                    >
                      {t('header.login')}
                    </Button>
                    <Button
                      onClick={() => {
                        setShowSidebar(false);
                        navigate('/register');
                      }}
                    >
                      {t('header.register')}
                    </Button>
                  </>
                )}
              </Menu.Item>
            </Menu>
          </Segment>
        ) : (
          <></>
        )}
      </>
    );
  }

  return (
    <>
      <Menu
        borderless
        style={{
          borderTop: 'none',
          boxShadow: 'rgba(0, 0, 0, 0.04) 0px 2px 12px 0px',
          border: 'none',
        }}
      >
        <Container
          style={{
            width: '100%',
            maxWidth: isMobile() ? '100%' : '1200px',
            padding: isMobile() ? '0 10px' : '0 20px',
          }}
        >
          <Menu.Item as={Link} to='/' className={'hide-on-mobile'}>
            <img src={logo} alt='logo' style={{ marginRight: '0.75em' }} />
            <div
              style={{
                fontSize: '18px',
                fontWeight: '500',
                color: '#333',
              }}
            >
              {systemName}
            </div>
          </Menu.Item>
          {renderButtons(false)}
          <Menu.Menu position='right'>
            <Menu.Item fitted='horizontally'>
              <Button
                type='button'
                basic
                icon
                style={{ margin: '0 4px' }}
                onClick={toggleLanguage}
                title={t('header.language_switch_tooltip')}
                aria-label={t('header.language_switch_tooltip')}
              >
                <Icon name='language' />
              </Button>
            </Menu.Item>
            {userState.user ? (
              <Dropdown
                text={userState.user.username}
                pointing
                className='link item'
                style={{
                  fontSize: '15px',
                  fontWeight: '400',
                  color: '#666',
                }}
              >
                <Dropdown.Menu>
                  <Dropdown.Item
                    onClick={logout}
                    style={{
                      fontSize: '15px',
                      fontWeight: '400',
                      color: '#666',
                    }}
                  >
                    {t('header.logout')}
                  </Dropdown.Item>
                </Dropdown.Menu>
              </Dropdown>
            ) : (
              <Menu.Item
                name={t('header.login')}
                as={Link}
                to='/login'
                className='btn btn-link'
                style={{
                  fontSize: '15px',
                  fontWeight: '400',
                  color: '#666',
                }}
              />
            )}
          </Menu.Menu>
        </Container>
      </Menu>
    </>
  );
};

export default Header;
