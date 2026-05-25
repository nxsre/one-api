/** 侧栏 / 登录 / favicon 共用的三层叠放图形 */
export const LOGO_MARK_PATH =
  'M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5';

export default function LogoMark({ size = 18, className, fill = 'currentColor' }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={fill}
      aria-hidden
    >
      <path d={LOGO_MARK_PATH} />
    </svg>
  );
}
