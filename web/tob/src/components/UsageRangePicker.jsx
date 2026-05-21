import { DatePicker, Segmented } from 'antd';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { RANGE_PRESETS } from '@/lib/usageReports';

dayjs.locale('zh-cn');

const { RangePicker } = DatePicker;

export default function UsageRangePicker({
  preset,
  startDate,
  endDate,
  loading,
  onPresetChange,
  onCustomRangeChange,
}) {
  const customRange =
    startDate && endDate ? [dayjs(startDate, 'YYYY-MM-DD'), dayjs(endDate, 'YYYY-MM-DD')] : null;

  return (
    <div className="usage-range-toolbar">
      <Segmented
        className="usage-range-segmented"
        options={RANGE_PRESETS.map((p) => ({ label: p.label, value: p.key }))}
        value={preset}
        disabled={loading}
        onChange={onPresetChange}
      />
      {preset === 'custom' ? (
        <RangePicker
          className="usage-range-picker"
          value={customRange}
          disabled={loading}
          allowClear={false}
          format="YYYY-MM-DD"
          onChange={(dates) => {
            if (!dates?.[0] || !dates?.[1]) return;
            onCustomRangeChange(
              dates[0].format('YYYY-MM-DD'),
              dates[1].format('YYYY-MM-DD')
            );
          }}
        />
      ) : null}
    </div>
  );
}
