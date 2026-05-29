<template>
  <template v-if="aboutLoaded && about === ''">
    <div class="dashboard-container">
      <a-card class="chart-card">
        <div class="header" style="font-size: 1.2em; font-weight: 600; margin-bottom: 15px;">
          {{ t('about.title') }}
        </div>
        <p>{{ t('about.description') }}</p>
        {{ t('about.repository') }}
        <a href="https://github.com/songquanpeng/one-api">https://github.com/songquanpeng/one-api</a>
        <p v-if="versionLine" :style="versionBannerStyle">{{ versionLine }}</p>
      </a-card>
    </div>
  </template>

  <template v-else>
    <template v-if="about.startsWith('https://')">
      <div v-if="versionLine" :style="iframeTopBarStyle">{{ versionLine }}</div>
      <iframe
        :src="about"
        title="about-external"
        style="width: 100%; height: 100vh; border: none;"
      />
    </template>
    <div v-else class="dashboard-container">
      <a-card class="chart-card">
        <div style="font-size: larger;" v-html="about"></div>
        <p v-if="versionLine" :style="versionBannerStyle">{{ versionLine }}</p>
      </a-card>
    </div>
  </template>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { marked } from 'marked';
import { API, showError } from '@/helpers';

const { t } = useI18n();

const about = ref('');
const aboutLoaded = ref(false);
const versions = ref(null);

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
    const base = (import.meta.env.BASE_URL || '').replace(/\/$/, '');
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

const displayAbout = async () => {
  about.value = localStorage.getItem('about') || '';
  const res = await API.get('/api/about');
  const { success, message, data } = res.data;
  if (success) {
    let aboutContent = data;
    if (!data.startsWith('https://')) {
      aboutContent = marked.parse(data);
    }
    about.value = aboutContent;
    localStorage.setItem('about', aboutContent);
  } else {
    showError(message);
    about.value = t('about.loading_failed');
  }
  aboutLoaded.value = true;
};

const versionLine = computed(() =>
  versions.value
    ? t('about.build_versions', {
        front_pkg: versions.value.frontendPkg || '—',
        front_build: versions.value.frontendBuild || '—',
        back_ver: versions.value.backendVersion || '—',
        back_build: versions.value.backendBuildId || '—',
      })
    : ''
);

const versionBannerStyle = {
  fontSize: '0.9rem',
  color: 'rgba(0,0,0,0.55)',
  marginTop: '12px',
  marginBottom: 0,
};

const iframeTopBarStyle = {
  padding: '10px 24px',
  fontSize: '0.9rem',
  color: '#555',
  borderBottom: '1px solid #eee',
  background: '#fafafa',
};

onMounted(() => {
  displayAbout();
  fetchRuntimeVersions(API).then((v) => {
    versions.value = v;
  });
});
</script>
