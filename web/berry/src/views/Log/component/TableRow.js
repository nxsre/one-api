import PropTypes from 'prop-types';

import { TableRow, TableCell } from '@mui/material';

import { timestamp2string, renderQuota } from 'utils/common';
import Label from 'ui-component/Label';
import LogType from '../type/LogType';

function renderType(type) {
  const typeOption = LogType[type];
  if (typeOption) {
    return (
      <Label variant="filled" color={typeOption.color}>
        {' '}
        {typeOption.text}{' '}
      </Label>
    );
  } else {
    return (
      <Label variant="filled" color="error">
        {' '}
        未知{' '}
      </Label>
    );
  }
}

export default function LogTableRow({ item, userIsAdmin }) {
  const otherPreview =
    item.other && String(item.other).trim() !== ''
      ? String(item.other).length > 160
        ? `${String(item.other).slice(0, 160)}…`
        : String(item.other)
      : '';
  return (
    <>
      <TableRow tabIndex={item.id}>
        <TableCell>{timestamp2string(item.created_at)}</TableCell>

        {userIsAdmin && <TableCell>{item.channel || ''}</TableCell>}
        {userIsAdmin && (
          <TableCell>
            <span style={{ wordBreak: 'break-all' }}>{item.channel_name || ''}</span>
          </TableCell>
        )}
        <TableCell>{item.group || ''}</TableCell>
        {userIsAdmin && (
          <TableCell>
            <Label color="default" variant="outlined">
              {item.username}
            </Label>
          </TableCell>
        )}
        <TableCell>
          {item.token_name && (
            <Label color="default" variant="soft">
              {item.token_name}
            </Label>
          )}
        </TableCell>
        <TableCell>{item.token_id || ''}</TableCell>
        <TableCell>
          {item.ip ? <code style={{ fontSize: '0.85em' }}>{item.ip}</code> : ''}
        </TableCell>
        <TableCell>{renderType(item.type)}</TableCell>
        <TableCell>
          {item.model_name && (
            <Label color="primary" variant="outlined">
              {item.model_name}
            </Label>
          )}
        </TableCell>
        <TableCell>{item.prompt_tokens || ''}</TableCell>
        <TableCell>{item.completion_tokens || ''}</TableCell>
        <TableCell>{item.quota ? renderQuota(item.quota, 6) : ''}</TableCell>
        <TableCell>{item.use_time || ''}</TableCell>
        <TableCell>
          {item.content}
          {otherPreview && (
            <>
              <br />
              <span style={{ fontSize: '0.8em', wordBreak: 'break-all' }}>{otherPreview}</span>
            </>
          )}
        </TableCell>
      </TableRow>
    </>
  );
}

LogTableRow.propTypes = {
  item: PropTypes.object,
  userIsAdmin: PropTypes.bool
};
