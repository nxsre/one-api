import React, { useEffect, useState } from 'react';
import { Header, Segment } from 'semantic-ui-react';
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
      setAbout('加载关于内容失败...');
    }
    setAboutLoaded(true);
  };

  useEffect(() => {
    displayAbout().then();
    fetchRuntimeVersions(API).then(setVersions);
  }, []);

  const versionLine =
    versions &&
    `构建版本：前端包 ${versions.frontendPkg || '—'}（构建 ${versions.frontendBuild || '—'}），后端 ${versions.backendVersion || '—'}（构建 ${versions.backendBuildId || '—'}）`;

  const bannerStyle = {
    fontSize: '0.9rem',
    color: 'rgba(0,0,0,0.55)',
    marginTop: 12,
  };

  const iframeBarStyle = {
    padding: '10px 16px',
    fontSize: '0.9rem',
    color: '#555',
    borderBottom: '1px solid #eee',
    background: '#fafafa',
  };

  return (
    <>
      {aboutLoaded && about === '' ? (
        <>
          <Segment>
            <Header as='h3'>关于</Header>
            <p>可在设置页面设置关于内容，支持 HTML & Markdown</p>
            项目仓库地址：
            <a href='https://github.com/songquanpeng/one-api'>
              https://github.com/songquanpeng/one-api
            </a>
            {versionLine ? <p style={bannerStyle}>{versionLine}</p> : null}
          </Segment>
        </>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <>
              {versionLine ? <div style={iframeBarStyle}>{versionLine}</div> : null}
              <iframe
                title='about'
                src={about}
                style={{ width: '100%', height: '100vh', border: 'none' }}
              />
            </>
          ) : (
            <div style={{ fontSize: 'larger' }}>
              <div dangerouslySetInnerHTML={{ __html: about }}></div>
              {versionLine ? <p style={bannerStyle}>{versionLine}</p> : null}
            </div>
          )}
        </>
      )}
    </>
  );
};

export default About;
