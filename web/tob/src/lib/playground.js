import { getApiErrorMessage } from '@/api/client';
import { fetchModelsPage } from '@/lib/modelCatalog';
import { fetchTokenById, fetchTokenPage, getCopyKeyValue } from '@/lib/tokens';

const relayBase = import.meta.env.VITE_API_BASE || '';

/** 语言模型列表（体验中心模型选择） */
export async function fetchPlaygroundLanguageModels() {
  const { items } = await fetchModelsPage({
    page: 1,
    pageSize: 100,
    filterKey: 'language',
    search: 'gpt-4o',
  });
  return (items || [])
    .filter((row) => String(row.model_id || '').trim() === 'gpt-4o')
    .map((row) => ({
      id: row.model_id,
      label: row.model_name || row.model_id,
    }))
    .filter((m) => m.id);
}

/** 可用 API Key（status=1 活跃） */
export async function fetchPlaygroundTokens() {
  const list = await fetchTokenPage(0, '');
  return list
    .filter((t) => Number(t.status) === 1)
    .map((t) => ({
      id: t.id,
      label: t.name || `Key #${t.id}`,
    }));
}

export async function resolveTokenKey(tokenId) {
  const data = await fetchTokenById(tokenId);
  const key = getCopyKeyValue(data?.key);
  if (!key) throw new Error('无法获取 API Key，请重新选择');
  return key;
}

function parseSseChatDelta(line) {
  const trimmed = line.trim();
  if (!trimmed.startsWith('data:')) return null;
  const data = trimmed.slice(5).trim();
  if (!data || data === '[DONE]') return null;
  try {
    const parsed = JSON.parse(data);
    const err = parsed?.error?.message;
    if (err) throw new Error(err);
    const delta = parsed?.choices?.[0]?.delta?.content;
    return typeof delta === 'string' ? delta : '';
  } catch (e) {
    if (e instanceof Error && e.message && !/JSON/i.test(e.message)) throw e;
    return null;
  }
}

async function readHttpErrorMessage(res) {
  const text = await res.text();
  try {
    const json = JSON.parse(text);
    return json?.error?.message || json?.message || text;
  } catch {
    return text || `请求失败 (${res.status})`;
  }
}

/**
 * 流式对话（OpenAI 兼容 SSE /v1/chat/completions）
 * @param {(chunk: string) => void} onDelta 每收到一段文本回调
 */
export async function streamChatCompletion({ apiKey, model, messages, signal, onDelta }) {
  const url = `${relayBase}/v1/chat/completions`;
  let res;
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      body: JSON.stringify({ model, messages, stream: true }),
      signal,
    });
  } catch (e) {
    if (e?.name === 'AbortError') throw e;
    throw new Error(getApiErrorMessage(e));
  }

  if (!res.ok) {
    throw new Error(await readHttpErrorMessage(res));
  }
  if (!res.body) {
    throw new Error('浏览器不支持流式响应');
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let hasContent = false;

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n');
      buffer = parts.pop() ?? '';
      for (const line of parts) {
        const piece = parseSseChatDelta(line);
        if (piece) {
          hasContent = true;
          onDelta(piece);
        }
      }
    }
    if (buffer.trim()) {
      const piece = parseSseChatDelta(buffer);
      if (piece) {
        hasContent = true;
        onDelta(piece);
      }
    }
  } finally {
    reader.releaseLock?.();
  }

  if (!hasContent) {
    throw new Error('模型未返回有效内容');
  }
}
