const fs = require('fs');
const path = require('path');
const axios = require('axios');

// 配置文件路径
const CONFIG_FILE = path.join(__dirname, 'config.json');

/**
 * 读取配置文件
 */
function readConfig() {
  try {
    if (fs.existsSync(CONFIG_FILE)) {
      const data = fs.readFileSync(CONFIG_FILE, 'utf8');
      return JSON.parse(data);
    }
  } catch (error) {
    console.error('读取配置文件失败:', error.message);
  }
  return {};
}

/**
 * 保存配置文件
 */
function saveConfig(config) {
  try {
    fs.writeFileSync(CONFIG_FILE, JSON.stringify(config, null, 2), 'utf8');
    console.log('配置已保存到:', CONFIG_FILE);
    return true;
  } catch (error) {
    console.error('保存配置文件失败:', error.message);
    return false;
  }
}

/**
 * 获取高德 Web Service Key
 */
function getWebServiceKey() {
  const config = readConfig();
  return config.webServiceKey || null;
}

/**
 * 设置高德 Web Service Key
 */
function setWebServiceKey(key) {
  const config = readConfig();
  config.webServiceKey = key;
  return saveConfig(config);
}

/**
 * 检查并提示用户输入 Key
 */
async function ensureWebServiceKey() {
  let key = process.env.AMAP_KEY || process.env.AMAP_WEBSERVICE_KEY;

  if (!key) {
    key = getWebServiceKey();
  }

  if (!key) {
    console.log('\n⚠️  未找到高德 Web Service Key');
    console.log('请访问以下地址创建应用并获取 Key:');
    console.log('https://lbs.amap.com/api/webservice/create-project-and-key\n');
    throw new Error('请设置环境变量 AMAP_KEY 或提供高德 Web Service Key');
  }

  return key;
}

function trimStr(v) {
  return v == null ? '' : String(v).trim();
}

/** @returns {{ base: string, token: string } | null} */
function readOneApiCredentials() {
  const config = readConfig();
  const base = trimStr(process.env.ONE_API_BASE || config.oneApiBase);
  const token = trimStr(process.env.ONE_API_TOKEN || config.oneApiToken);
  if (!base || !token) return null;
  return { base: base.replace(/\/+$/, ''), token };
}

function shouldUseOneApiForPOI() {
  return readOneApiCredentials() !== null;
}

/** 统一 pois 为数组（兼容高德 v5 返回 pois 或 pois.poi） */
function normalizeGaodePOIResponse(data) {
  if (!data || typeof data !== 'object') return null;
  const out = { ...data };
  let pois = out.pois;
  if (pois && !Array.isArray(pois) && Array.isArray(pois.poi)) {
    pois = pois.poi;
  }
  out.pois = Array.isArray(pois) ? pois : [];
  return out;
}

function buildAxiosOptionsForOneApi() {
  /** @type {import('axios').AxiosRequestConfig} */
  const opts = { timeout: 60000 };
  const insecure =
    process.env.ONE_API_INSECURE === '1' ||
    process.env.ONE_API_INSECURE === 'true' ||
    process.env.ONE_API_SKIP_TLS_VERIFY === '1';
  if (insecure) {
    const https = require('https');
    opts.httpsAgent = new https.Agent({ rejectUnauthorized: false });
  }
  return opts;
}

/**
 * 经 one-api POST /amap 转发高德 v5/place（须 ONE_API_BASE + ONE_API_TOKEN）。
 */
async function searchPOIViaOneApi(params) {
  const cred = readOneApiCredentials();
  if (!cred) {
    throw new Error('缺少 ONE_API_BASE + ONE_API_TOKEN（或 config.json 中 oneApiBase / oneApiToken）');
  }

  /** @type {{ method: string, path: string, query: Record<string, string> }} */
  const proxyBody = {
    method: 'GET',
    path: '/v5/place/text',
    query: {},
  };

  const q = proxyBody.query;
  q.page_num = String(params.page || 1);
  q.page_size = String(params.offset || 10);
  if (params.types && String(params.types).trim()) {
    q.types = String(params.types).trim();
  }

  if (params.location && String(params.location).trim()) {
    proxyBody.path = '/v5/place/around';
    q.location = String(params.location).trim();
    q.keywords = params.keywords != null ? String(params.keywords) : '';
    q.radius = params.radius != null ? String(params.radius) : '1000';
    if (params.city && String(params.city).trim()) {
      q.region = String(params.city).trim();
    }
    q.city_limit = params.cityLimit !== false ? 'true' : 'false';
  } else {
    proxyBody.path = '/v5/place/text';
    q.keywords = params.keywords != null ? String(params.keywords) : '';
    if (params.city && String(params.city).trim()) {
      q.region = String(params.city).trim();
    }
    q.city_limit = params.cityLimit !== false ? 'true' : 'false';
    const hasKw = q.keywords && q.keywords.trim() !== '';
    const hasTypes = q.types && q.types.trim() !== '';
    if (!hasKw && !hasTypes) {
      console.error('❌ 关键字搜索须至少提供 keywords 或 types');
      return null;
    }
  }

  const url = `${cred.base}/amap`;
  console.log('🔍 正在通过 one-api POST /amap 搜索 POI…');

  try {
    const axiosOpts = {
      ...buildAxiosOptionsForOneApi(),
      headers: {
        Authorization: `Bearer ${cred.token}`,
        'Content-Type': 'application/json',
      },
    };
    const response = await axios.post(url, proxyBody, axiosOpts);

    const data = response.data;

    if (data && typeof data === 'object' && data.error) {
      const msg = data.error.message || JSON.stringify(data.error);
      console.error('❌ one-api 错误:', msg);
      return null;
    }

    if (response.status !== 200) {
      console.error('❌ one-api HTTP', response.status, data != null ? JSON.stringify(data).slice(0, 400) : '');
      return null;
    }

    const normalized = normalizeGaodePOIResponse(data);
    if (!normalized) return null;

    if (normalized.status === '1') {
      console.log(`✅ 搜索成功，共找到 ${normalized.count != null ? normalized.count : normalized.pois.length} 条结果\n`);
      return normalized;
    }
    console.error('❌ 高德返回失败:', normalized.info || normalized);
    return null;
  } catch (error) {
    const msg =
      error.response && error.response.data
        ? `${error.message} — ${JSON.stringify(error.response.data).slice(0, 500)}`
        : error.message;
    console.error('❌ 请求失败:', msg);
    return null;
  }
}

/**
 * POI 搜索（仅经 one-api POST /amap，须配置 ONE_API_BASE + ONE_API_TOKEN）
 * @param {Object} params - 搜索参数
 * @param {string} params.keywords - 查询关键字
 * @param {string} params.city - 城市名称或城市编码
 * @param {string} params.types - POI类型编码
 * @param {string} params.location - 中心点坐标
 * @param {number} params.radius - 搜索半径(米)
 * @param {number} params.page - 当前页数
 * @param {number} params.offset - 每页记录数
 */
async function searchPOI(params) {
  return searchPOIViaOneApi(params);
}

/**
 * 步行路径规划
 */
async function walkingRoute(params) {
  const key = await ensureWebServiceKey();

  const url = 'https://restapi.amap.com/v3/direction/walking';

  const requestParams = {
    key: key,
    origin: params.origin,
    destination: params.destination,
    appname: 'amap-lbs-skill',
  };

  try {
    console.log('🚶 正在规划步行路线...');
    const response = await axios.get(url, { params: requestParams });

    if (response.data.status === '1') {
      console.log('✅ 步行路线规划成功\n');
      return response.data;
    } else {
      console.error('❌ 步行路线规划失败:', response.data.info);
      return null;
    }
  } catch (error) {
    console.error('❌ 请求失败:', error.message);
    return null;
  }
}

/**
 * 驾车路径规划
 */
async function drivingRoute(params) {
  const key = await ensureWebServiceKey();

  const url = 'https://restapi.amap.com/v3/direction/driving';

  const requestParams = {
    key: key,
    origin: params.origin,
    destination: params.destination,
    strategy: params.strategy || 10,
    extensions: 'base',
    appname: 'amap-lbs-skill',
  };

  if (params.waypoints) {
    requestParams.waypoints = params.waypoints;
  }

  try {
    console.log('🚗 正在规划驾车路线...');
    const response = await axios.get(url, { params: requestParams });

    if (response.data.status === '1') {
      console.log('✅ 驾车路线规划成功\n');
      return response.data;
    } else {
      console.error('❌ 驾车路线规划失败:', response.data.info);
      return null;
    }
  } catch (error) {
    console.error('❌ 请求失败:', error.message);
    return null;
  }
}

/**
 * 骑行路径规划
 */
async function ridingRoute(params) {
  const key = await ensureWebServiceKey();

  const url = 'https://restapi.amap.com/v4/direction/bicycling';

  const requestParams = {
    key: key,
    origin: params.origin,
    destination: params.destination,
    appname: 'amap-lbs-skill',
  };

  try {
    console.log('🚴 正在规划骑行路线...');
    const response = await axios.get(url, { params: requestParams });

    if (response.data.errcode === 0) {
      console.log('✅ 骑行路线规划成功\n');
      return response.data;
    } else {
      console.error('❌ 骑行路线规划失败:', response.data.errmsg);
      return null;
    }
  } catch (error) {
    console.error('❌ 请求失败:', error.message);
    return null;
  }
}

/**
 * 公交路径规划
 */
async function transitRoute(params) {
  const key = await ensureWebServiceKey();

  const url = 'https://restapi.amap.com/v3/direction/transit/integrated';

  const requestParams = {
    key: key,
    origin: params.origin,
    destination: params.destination,
    city: params.city,
    strategy: params.strategy || 0,
    nightflag: params.nightflag ? 1 : 0,
    appname: 'amap-lbs-skill',
  };

  try {
    console.log('🚌 正在规划公交路线...');
    const response = await axios.get(url, { params: requestParams });

    if (response.data.status === '1') {
      console.log('✅ 公交路线规划成功\n');
      return response.data;
    } else {
      console.error('❌ 公交路线规划失败:', response.data.info);
      return null;
    }
  } catch (error) {
    console.error('❌ 请求失败:', error.message);
    return null;
  }
}

/**
 * 生成地图可视化链接
 * @param {Array} mapTaskData - 地图任务数据数组
 * @returns {string} 可视化链接
 */
function generateMapLink(mapTaskData) {
  const baseUrl = 'https://a.amap.com/jsapi_demo_show/static/openclaw/travel_plan.html';
  const dataStr = encodeURIComponent(JSON.stringify(mapTaskData));
  return `${baseUrl}?data=${dataStr}`;
}

/**
 * 旅游规划助手
 * @param {Object} params - 规划参数
 * @param {string} params.city - 城市名称
 * @param {Array<string>} params.interests - 兴趣点关键词数组，如 ['景点', '美食', '酒店']
 * @param {string} params.routeType - 路线类型：driving/walking/riding/transfer
 * @returns {Object} 包含 pois、mapTaskData
 */
async function travelPlanner(params) {
  const { city, interests = [], routeType = 'walking' } = params;
  
  console.log(`\n🗺️  开始为您规划 ${city} 的旅游行程...\n`);
  
  const mapTaskData = [];
  const poiResults = [];
  
  // 搜索各类兴趣点
  for (const interest of interests) {
    console.log(`📍 搜索 ${interest}...`);
    const result = await searchPOI({
      keywords: interest,
      city: city,
      page: 1,
      offset: 5
    });
    
    if (result && result.pois && result.pois.length > 0) {
      poiResults.push(...result.pois);
      
      // 添加到地图数据 - 严格按照 PoiTask 接口格式
      result.pois.forEach(poi => {
        const [lng, lat] = poi.location.split(',').map(Number);
        mapTaskData.push({
          type: 'poi',
          lnglat: [lng, lat],
          sort: poi.type || interest,
          text: poi.name,
          remark: poi.address || `${interest}推荐`
        });
      });
    }
  }
  
  // 如果有多个POI，规划路线
  if (poiResults.length >= 2) {
    console.log(`\n🛣️  规划游览路线（${routeType}）...\n`);
    
    for (let i = 0; i < poiResults.length - 1; i++) {
      const start = poiResults[i];
      const end = poiResults[i + 1];
      
      const [startLng, startLat] = start.location.split(',').map(Number);
      const [endLng, endLat] = end.location.split(',').map(Number);
      
      // 添加路线到地图数据 - 严格按照 RouteTask 接口格式
      const routeTask = {
        type: 'route',
        routeType: routeType,
        start: [startLng, startLat],
        end: [endLng, endLat],
        remark: `从 ${start.name} 到 ${end.name}`
      };
      
      // 如果是公交路线，添加 city 参数
      if (routeType === 'transfer') {
        routeTask.city = city;
      }
      
      mapTaskData.push(routeTask);
    }
  }
  
  
  console.log('\n✅ 旅游规划完成！\n');
  console.log('📍 推荐地点：');
  poiResults.forEach((poi, index) => {
    console.log(`${index + 1}. ${poi.name}`);
    console.log(`   地址: ${poi.address}`);
    console.log(`   类型: ${poi.type}\n`);
  });

  return {
    pois: poiResults,
    mapTaskData,
  };
}

// 导出函数供其他脚本使用
module.exports = {
  readConfig,
  saveConfig,
  getWebServiceKey,
  setWebServiceKey,
  ensureWebServiceKey,
  shouldUseOneApiForPOI,
  searchPOI,
  walkingRoute,
  drivingRoute,
  ridingRoute,
  transitRoute,
  generateMapLink,
  travelPlanner,
};

// 如果直接运行此文件，执行示例搜索
if (require.main === module) {
  (async () => {
    try {
      // 示例：搜索北京的肯德基
      const result = await searchPOI({
        keywords: '肯德基',
        city: '北京',
        page: 1,
        offset: 10
      });
      
      if (result && result.pois) {
        console.log('搜索结果:');
        result.pois.forEach((poi, index) => {
          console.log(`${index + 1}. ${poi.name}`);
          console.log(`   地址: ${poi.address}`);
          console.log(`   类型: ${poi.type}`);
          console.log(`   坐标: ${poi.location}\n`);
        });
      }
    } catch (error) {
      console.error('执行失败:', error.message);
      process.exit(1);
    }
  })();
}
