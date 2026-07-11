import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

export default function MarkdownContent({ content, streaming = false }) {
  if (!content) {
    if (streaming) {
      return <span className="playground-stream-placeholder">正在思考…</span>;
    }
    return null;
  }

  return (
    <div className={`playground-md${streaming ? ' playground-md--streaming' : ''}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noopener noreferrer">
              {children}
            </a>
          ),
          pre: ({ children }) => <pre className="playground-md-pre">{children}</pre>,
          code: ({ className, children, ...props }) => {
            const isBlock = Boolean(className);
            return (
              <code
                className={isBlock ? `playground-md-code-block ${className || ''}` : 'playground-md-code-inline'}
                {...props}
              >
                {children}
              </code>
            );
          },
          table: ({ children }) => (
            <div className="playground-md-table-wrap">
              <table className="playground-md-table">{children}</table>
            </div>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
      {streaming ? <span className="playground-stream-cursor" aria-hidden /> : null}
    </div>
  );
}
