import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from 'semantic-ui-react';
import { API, showError } from '../../helpers';
import { marked } from 'marked';

async function fetchRuntimeVersions(api) {
  let backendVersion = '';
  let backendBuildId = '';
  try {
    const res = await api.get('/api/status');
    const d = res.data?.data;
    if (res.data?.success && d) {
      backendVersion = d.version != null ? String(d.version) : '';
      backendBuildId = d.build_id != null ? String(d.build_id) : '';
    }
  } catch (_) {
    /* ignore */
  }

  let frontendPkg = '';
  let frontendBuild = '';
  try {
    const base = (process.env.PUBLIC_URL || '').replace(/\/$/, '');
    const r = await fetch(`${base}/web-build.json`);
    if (r.ok) {
      const j = await r.json();
      frontendPkg = j.package_version != null ? String(j.package_version) : '';
      frontendBuild = j.build_id != null ? String(j.build_id) : '';
    }
  } catch (_) {
    /* ignore */
  }

  return { backendVersion, backendBuildId, frontendPkg, frontendBuild };
}

const About = () => {
  const { t } = useTranslation();
  const [about, setAbout] = useState('');
  const [aboutLoaded, setAboutLoaded] = useState(false);
  const [versions, setVersions] = useState(null);

  const displayAbout = async () => {
    setAbout(localStorage.getItem('about') || '');
    const res = await API.get('/api/about');
    const { success, message, data } = res.data;
    if (success) {
      let aboutContent = data;
      if (!data.startsWith('https://')) {
        aboutContent = marked.parse(data);
      }
      setAbout(aboutContent);
      localStorage.setItem('about', aboutContent);
    } else {
      showError(message);
      setAbout(t('about.loading_failed'));
    }
    setAboutLoaded(true);
  };

  useEffect(() => {
    displayAbout().then();
    fetchRuntimeVersions(API).then(setVersions);
  }, []);

  const versionLine =
    versions &&
    t('about.build_versions', {
      front_pkg: versions.frontendPkg || '—',
      front_build: versions.frontendBuild || '—',
      back_ver: versions.backendVersion || '—',
      back_build: versions.backendBuildId || '—',
    });

  const versionBannerStyle = {
    fontSize: '0.9rem',
    color: 'rgba(0,0,0,0.55)',
    marginTop: 12,
    marginBottom: 0,
  };

  const iframeTopBarStyle = {
    padding: '10px 24px',
    fontSize: '0.9rem',
    color: '#555',
    borderBottom: '1px solid #eee',
    background: '#fafafa',
  };

  return (
    <>
      {aboutLoaded && about === '' ? (
        <div className='dashboard-container'>
          <Card fluid className='chart-card'>
            <Card.Content>
              <Card.Header className='header'>{t('about.title')}</Card.Header>
              <p>{t('about.description')}</p>
              {t('about.repository')}
              <a href='https://github.com/songquanpeng/one-api'>
                https://github.com/songquanpeng/one-api
              </a>
              {versionLine ? <p style={versionBannerStyle}>{versionLine}</p> : null}
            </Card.Content>
          </Card>
        </div>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <>
              {versionLine ? (
                <div style={iframeTopBarStyle}>{versionLine}</div>
              ) : null}
              <iframe
                src={about}
                title='about-external'
                style={{ width: '100%', height: '100vh', border: 'none' }}
              />
            </>
          ) : (
            <div className='dashboard-container'>
              <Card fluid className='chart-card'>
                <Card.Content>
                  <div
                    style={{ fontSize: 'larger' }}
                    dangerouslySetInnerHTML={{ __html: about }}
                  ></div>
                  {versionLine ? <p style={versionBannerStyle}>{versionLine}</p> : null}
                </Card.Content>
              </Card>
            </div>
          )}
        </>
      )}
    </>
  );
};

export default About;
