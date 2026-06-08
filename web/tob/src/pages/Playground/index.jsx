import { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Select, message } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import {
  fetchPlaygroundLanguageModels,
  fetchPlaygroundTokens,
  streamChatCompletion,
  resolveTokenKey,
} from '@/lib/playground';
import MarkdownContent from './MarkdownContent';
import './playground.css';

function newMessage(role, content) {
  return { id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2)}`, role, content };
}

export default function PlaygroundPage() {
  const [models, setModels] = useState([]);
  const [tokens, setTokens] = useState([]);
  const [modelId, setModelId] = useState('');
  const [tokenId, setTokenId] = useState(null);
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');
  const [loadingMeta, setLoadingMeta] = useState(true);
  const [sending, setSending] = useState(false);
  const [streamingId, setStreamingId] = useState(null);
  const listRef = useRef(null);
  const abortRef = useRef(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoadingMeta(true);
      try {
        const [modelList, tokenList] = await Promise.all([
          fetchPlaygroundLanguageModels(),
          fetchPlaygroundTokens(),
        ]);
        if (cancelled) return;
        setModels(modelList);
        setTokens(tokenList);
        setModelId(modelList[0]?.id || '');
        setTokenId(tokenList[0]?.id ?? null);
      } catch (e) {
        if (!cancelled) message.error(e?.message || '加载失败');
      } finally {
        if (!cancelled) setLoadingMeta(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages, sending, streamingId]);

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || sending) return;
    if (!modelId) {
      message.warning('请选择模型');
      return;
    }
    if (!tokenId) {
      message.warning('请先在 API KEY 中创建并启用密钥');
      return;
    }

    const userMsg = newMessage('user', text);
    const nextMessages = [...messages, userMsg];
    setMessages(nextMessages);
    setInput('');
    setSending(true);

    const controller = new AbortController();
    abortRef.current = controller;

    const assistantMsg = newMessage('assistant', '');
    setMessages((prev) => [...prev, assistantMsg]);
    setStreamingId(assistantMsg.id);

    try {
      const apiKey = await resolveTokenKey(tokenId);
      const apiMessages = nextMessages.map((m) => ({
        role: m.role,
        content: m.content,
      }));
      await streamChatCompletion({
        apiKey,
        model: modelId,
        messages: apiMessages,
        signal: controller.signal,
        onDelta: (chunk) => {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantMsg.id ? { ...m, content: m.content + chunk } : m
            )
          );
        },
      });
    } catch (e) {
      if (e?.name === 'AbortError' || /abort/i.test(e?.message || '')) {
        setMessages((prev) => prev.filter((m) => m.id !== assistantMsg.id));
        return;
      }
      message.error(e?.message || '请求失败');
      setMessages((prev) => prev.filter((m) => m.id !== assistantMsg.id));
      setInput(text);
    } finally {
      setSending(false);
      setStreamingId(null);
      abortRef.current = null;
    }
  }, [input, sending, modelId, tokenId, messages]);

  const onKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      void handleSend();
    }
  };

  const showWelcome = messages.length === 0 && !sending;

  return (
    <div className="playground-page">
      <div className="playground-toolbar">
        <Select
          className="playground-select"
          placeholder="选择模型"
          loading={loadingMeta}
          value={modelId || undefined}
          onChange={setModelId}
          options={models.map((m) => ({ value: m.id, label: m.label }))}
          showSearch
          optionFilterProp="label"
        />
        <Select
          className="playground-select playground-select--token"
          placeholder="选择 API Key"
          loading={loadingMeta}
          value={tokenId ?? undefined}
          onChange={setTokenId}
          options={tokens.map((t) => ({ value: t.id, label: t.label }))}
        />
        {tokens.length === 0 && !loadingMeta ? (
          <Link to="/api-keys" className="playground-link">
            去创建 API KEY
          </Link>
        ) : null}
      </div>

      <div className="playground-main" ref={listRef}>
        {showWelcome ? (
          <div className="playground-welcome">
            <div className="playground-welcome-logo" aria-hidden>
              ✦
            </div>
            <h2 className="playground-welcome-title">语言模型体验</h2>
            <p className="playground-welcome-desc">输入问题，模型将为你解答</p>
          </div>
        ) : (
          <div className="playground-messages">
            {messages.map((m) => (
              <div
                key={m.id}
                className={`playground-bubble playground-bubble--${m.role}`}
              >
                <div className="playground-bubble-role">
                  {m.role === 'user' ? '我' : '模型'}
                </div>
                <div className="playground-bubble-content">
                  {m.role === 'assistant' ? (
                    <MarkdownContent
                      content={m.content}
                      streaming={m.id === streamingId}
                    />
                  ) : (
                    m.content
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="playground-composer-wrap">
        <div className="playground-composer">
          <textarea
            className="playground-input"
            placeholder="请输入问题，帮你深度解答"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={onKeyDown}
            rows={3}
            disabled={sending}
          />
          <div className="playground-composer-bar">
            <span className="playground-hint">Enter 发送 · Shift+Enter 换行</span>
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={sending}
              disabled={!input.trim() || loadingMeta}
              onClick={() => void handleSend()}
            >
              发送
            </Button>
          </div>
        </div>
        <p className="playground-disclaimer">
          体验内容由人工智能模型生成，不代表平台立场
        </p>
      </div>
    </div>
  );
}
