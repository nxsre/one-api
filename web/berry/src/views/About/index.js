import React, { useEffect, useState } from 'react';
import { API } from 'utils/api';
import { showError } from 'utils/common';
import { marked } from 'marked';
import { Box, Container, Typography } from '@mui/material';
import MainCard from 'ui-component/cards/MainCard';

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

  return (
    <>
      {aboutLoaded && about === '' ? (
        <>
          <Box>
            <Container sx={{ paddingTop: '40px' }}>
              <MainCard title="关于">
                <Typography variant="body2">
                  可在设置页面设置关于内容，支持 HTML & Markdown <br />
                  项目仓库地址：
                  <a href="https://github.com/songquanpeng/one-api">https://github.com/songquanpeng/one-api</a>
                </Typography>
                {versionLine ? (
                  <Typography variant="caption" display="block" sx={{ mt: 2, color: 'text.secondary' }}>
                    {versionLine}
                  </Typography>
                ) : null}
              </MainCard>
            </Container>
          </Box>
        </>
      ) : (
        <>
          <Box>
            {about.startsWith('https://') ? (
              <>
                {versionLine ? (
                  <Box sx={{ px: 2, py: 1.25, fontSize: '0.875rem', color: 'text.secondary', borderBottom: '1px solid', borderColor: 'divider', bgcolor: 'grey.50' }}>
                    {versionLine}
                  </Box>
                ) : null}
                <iframe title="about" src={about} style={{ width: '100%', height: '100vh', border: 'none' }} />
              </>
            ) : (
              <>
                <Container>
                  <div style={{ fontSize: 'larger' }} dangerouslySetInnerHTML={{ __html: about }}></div>
                  {versionLine ? (
                    <Typography variant="caption" display="block" sx={{ mt: 2, color: 'text.secondary' }}>
                      {versionLine}
                    </Typography>
                  ) : null}
                </Container>
              </>
            )}
          </Box>
        </>
      )}
    </>
  );
};

export default About;
