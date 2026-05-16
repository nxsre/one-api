import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Confirm,
  Form,
  Loader,
  Modal,
  Table,
} from 'semantic-ui-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  getStoredNacosNamespace,
  setStoredNacosNamespace,
  showError,
  showSuccess,
} from '../../helpers';
import NacosNamespaceSelect from '../../components/NacosNamespaceSelect';
import SettingMonacoField from '../../components/SettingMonacoField';

const defaultPromptContent = () =>
  JSON.stringify({ messages: [{ role: 'system', content: 'You are a helpful assistant.' }] }, null, 2);

const NacosPromptsRegistry = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [namespace, setNamespace] = useState(() => getStoredNacosNamespace());

  const [headerOpen, setHeaderOpen] = useState(false);
  const [hk, setHk] = useState('');
  const [hDesc, setHDesc] = useState('');
  const [hBiz, setHBiz] = useState('');
  const [hScope, setHScope] = useState('PUBLIC');
  const [hEnable, setHEnable] = useState(true);
  const [hkLocked, setHkLocked] = useState(false);
  const [headerBaseline, setHeaderBaseline] = useState({
    k: '',
    desc: '',
    biz: '',
    scope: 'PUBLIC',
  });

  const [verOpen, setVerOpen] = useState(false);
  const [verKey, setVerKey] = useState('');
  const [verContent, setVerContent] = useState(defaultPromptContent());
  const [verBaseline, setVerBaseline] = useState(defaultPromptContent());

  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const [submitOpen, setSubmitOpen] = useState(false);
  const [submitKey, setSubmitKey] = useState('');
  const [submitVer, setSubmitVer] = useState('');
  const [submitVerBaseline, setSubmitVerBaseline] = useState('');

  const [publishOpen, setPublishOpen] = useState(false);
  const [pubKey, setPubKey] = useState('');
  const [pubVer, setPubVer] = useState('');
  const [pubVerBaseline, setPubVerBaseline] = useState('');
  const [pubLatest, setPubLatest] = useState(true);
  const [pubForce, setPubForce] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteKey, setDeleteKey] = useState('');

  const [labelsOpen, setLabelsOpen] = useState(false);
  const [labelsKey, setLabelsKey] = useState('');
  const [labelsText, setLabelsText] = useState('{}');
  const [labelsReplace, setLabelsReplace] = useState(false);
  const [labelsBaseline, setLabelsBaseline] = useState('{}');

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/nacos/prompts', {
        params: { namespace, page: 1, size: 200 },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'load failed');
        return;
      }
      setRows(res.data.data?.pageItems || []);
    } catch (e) {
      showError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setStoredNacosNamespace(namespace);
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespace]);

  const openHeaderCreate = () => {
    setHkLocked(false);
    setHk('');
    setHDesc('');
    setHBiz('');
    setHScope('PUBLIC');
    setHEnable(true);
    setHeaderBaseline({ k: '', desc: '', biz: '', scope: 'PUBLIC' });
    setHeaderOpen(true);
  };

  const saveHeader = async () => {
    const key = hk.trim();
    if (!key) {
      showError(t('nacos.prompts_key_required'));
      return;
    }
    try {
      const res = await API.post(
        `/api/nacos/prompts/header?namespace=${encodeURIComponent(namespace)}`,
        {
          promptKey: key,
          description: hDesc,
          bizTags: hBiz,
          scope: hScope,
          enable: hEnable,
        }
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'save failed');
        return;
      }
      showSuccess(t('nacos.prompts_header_saved'));
      setHeaderOpen(false);
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const saveVersion = async () => {
    let parsed;
    try {
      parsed = JSON.parse(verContent || '{}');
    } catch {
      showError(t('nacos.prompts_invalid_json'));
      return;
    }
    try {
      const res = await API.post(
        `/api/nacos/prompts/version?namespace=${encodeURIComponent(namespace)}`,
        { promptKey: verKey, content: parsed }
      );
      if (!res.data?.success) {
        showError(res.data?.message || 'add version failed');
        return;
      }
      showSuccess(t('nacos.prompts_version_added'));
      setVerOpen(false);
      load();
      if (detailOpen && detail?.promptKey === verKey) {
        openDetail(verKey);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const openDetail = async (key) => {
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(null);
    try {
      const res = await API.get('/api/nacos/prompts/detail', {
        params: { namespace, promptKey: key },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'detail failed');
        setDetailOpen(false);
        return;
      }
      setDetail(res.data.data);
    } catch (e) {
      showError(e.message);
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const doSubmit = async () => {
    try {
      const res = await API.post('/api/nacos/prompts/submit', {
        namespace,
        promptKey: submitKey,
        version: submitVer || undefined,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'submit failed');
        return;
      }
      showSuccess(t('nacos.prompts_submit_ok'));
      setSubmitOpen(false);
      setSubmitVer('');
      setSubmitVerBaseline('');
      load();
      if (detailOpen && detail?.promptKey === submitKey) {
        openDetail(submitKey);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const doPublish = async () => {
    if (!pubVer.trim()) {
      showError(t('nacos.prompts_version_required'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/prompts/publish', {
        namespace,
        promptKey: pubKey,
        version: pubVer.trim(),
        updateLatest: pubLatest,
        forcePublish: pubForce,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'publish failed');
        return;
      }
      showSuccess(t('nacos.prompts_publish_ok'));
      setPublishOpen(false);
      load();
      if (detailOpen && detail?.promptKey === pubKey) {
        openDetail(pubKey);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  const doDelete = async () => {
    try {
      const res = await API.delete('/api/nacos/prompts/item', {
        params: { namespace, promptKey: deleteKey },
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'delete failed');
        return;
      }
      showSuccess(t('nacos.prompts_deleted'));
      setDeleteOpen(false);
      setDeleteKey('');
      load();
    } catch (e) {
      showError(e.message);
    }
  };

  const saveLabels = async () => {
    let labels;
    try {
      labels = JSON.parse(labelsText || '{}');
    } catch {
      showError(t('nacos.prompts_labels_invalid'));
      return;
    }
    try {
      const res = await API.post('/api/nacos/prompts/labels', {
        namespace,
        promptKey: labelsKey,
        labels,
        replace: labelsReplace,
      });
      if (!res.data?.success) {
        showError(res.data?.message || 'save labels failed');
        return;
      }
      showSuccess(t('nacos.prompts_labels_saved'));
      setLabelsOpen(false);
      load();
      if (detailOpen && detail?.promptKey === labelsKey) {
        openDetail(labelsKey);
      }
    } catch (e) {
      showError(e.message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('nacos.prompts_title')}</Card.Header>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              columnGap: 12,
              rowGap: 8,
              marginBottom: 12,
            }}
          >
            <span style={{ flexShrink: 0, color: '#666' }}>namespace</span>
            <div
              style={{ flex: '1 1 220px', minWidth: 180, maxWidth: 420 }}
            >
              <NacosNamespaceSelect
                value={namespace}
                onChange={(v) => setNamespace(v || 'public')}
              />
            </div>
            <Button size='small' onClick={load}>
              {t('nacos.refresh')}
            </Button>
            <Button size='small' primary onClick={openHeaderCreate}>
              {t('nacos.prompts_new_header')}
            </Button>
          </div>
          {loading ? (
            <Loader active />
          ) : (
            <Table celled compact>
              <Table.Header>
                <Table.Row>
                  <Table.HeaderCell>{t('nacos.prompts_col_key')}</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.prompts_col_desc')}</Table.HeaderCell>
                  <Table.HeaderCell>editing</Table.HeaderCell>
                  <Table.HeaderCell>reviewing</Table.HeaderCell>
                  <Table.HeaderCell>online</Table.HeaderCell>
                  <Table.HeaderCell>{t('nacos.prompts_col_actions')}</Table.HeaderCell>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <Table.Row key={r.promptKey}>
                    <Table.Cell>{r.promptKey}</Table.Cell>
                    <Table.Cell>{r.description}</Table.Cell>
                    <Table.Cell>{r.editingVersion}</Table.Cell>
                    <Table.Cell>{r.reviewingVersion}</Table.Cell>
                    <Table.Cell>{r.onlineCnt}</Table.Cell>
                    <Table.Cell>
                      <Button size='mini' onClick={() => openDetail(r.promptKey)}>
                        {t('nacos.prompts_detail')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setHkLocked(true);
                          setHk(r.promptKey);
                          setHDesc(r.description || '');
                          setHBiz(r.bizTags || '');
                          setHScope(r.scope || 'PUBLIC');
                          setHEnable(!!r.enable);
                          setHeaderBaseline({
                            k: r.promptKey,
                            desc: r.description || '',
                            biz: r.bizTags || '',
                            scope: r.scope || 'PUBLIC',
                          });
                          setHeaderOpen(true);
                        }}
                      >
                        {t('nacos.prompts_edit_header')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setVerKey(r.promptKey);
                          const vc = defaultPromptContent();
                          setVerContent(vc);
                          setVerBaseline(vc);
                          setVerOpen(true);
                        }}
                      >
                        {t('nacos.prompts_add_version')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setSubmitKey(r.promptKey);
                          setSubmitVer('');
                          setSubmitVerBaseline('');
                          setSubmitOpen(true);
                        }}
                      >
                        {t('nacos.prompts_submit')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          setPubKey(r.promptKey);
                          const pv = r.reviewingVersion || '';
                          setPubVer(pv);
                          setPubVerBaseline(pv);
                          setPubLatest(true);
                          setPubForce(false);
                          setPublishOpen(true);
                        }}
                      >
                        {t('nacos.prompts_publish')}
                      </Button>
                      <Button
                        size='mini'
                        onClick={() => {
                          const lt = JSON.stringify(r.labels || {}, null, 2);
                          setLabelsKey(r.promptKey);
                          setLabelsText(lt);
                          setLabelsBaseline(lt);
                          setLabelsReplace(false);
                          setLabelsOpen(true);
                        }}
                      >
                        {t('nacos.prompts_action_labels')}
                      </Button>
                      <Button
                        size='mini'
                        negative
                        onClick={() => {
                          setDeleteKey(r.promptKey);
                          setDeleteOpen(true);
                        }}
                      >
                        {t('nacos.prompts_delete')}
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table>
          )}
        </Card.Content>
      </Card>

      <Modal open={headerOpen} onClose={() => setHeaderOpen(false)}>
        <Modal.Header>{t('nacos.prompts_header_title')}</Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.prompts_col_key')}
              value={hk}
              originValue={headerBaseline.k}
              onChange={(v) => setHk(v)}
              readOnly={hkLocked}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.prompts_col_desc')}
              value={hDesc}
              originValue={headerBaseline.desc}
              onChange={(v) => setHDesc(v)}
              height={140}
            />
            <SettingMonacoField
              label='bizTags'
              value={hBiz}
              originValue={headerBaseline.biz}
              onChange={(v) => setHBiz(v)}
              height={88}
            />
            <SettingMonacoField
              label={t('nacos.prompts_col_scope')}
              hint='PUBLIC / PRIVATE'
              value={hScope}
              originValue={headerBaseline.scope}
              onChange={(v) => setHScope(v)}
              height={88}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.prompts_col_enable')}
                checked={hEnable}
                onChange={(e, { checked }) => setHEnable(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setHeaderOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button primary onClick={saveHeader}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={verOpen} onClose={() => setVerOpen(false)} size='large'>
        <Modal.Header>
          {t('nacos.prompts_add_version')}: {verKey}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label='content (JSON)'
              language='json'
              enableJsonFormat
              minimap
              value={verContent}
              originValue={verBaseline}
              onChange={(v) => setVerContent(v)}
              height={400}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setVerOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button primary onClick={saveVersion}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={detailOpen} onClose={() => setDetailOpen(false)} size='large'>
        <Modal.Header>{t('nacos.prompts_detail_title')}</Modal.Header>
        <Modal.Content scrolling>
          {detailLoading ? (
            <Loader active />
          ) : detail ? (
            <>
              <p>
                <strong>{detail.promptKey}</strong> — {detail.description}
              </p>
              <p>
                enable: {detail.enable ? 'yes' : 'no'} | scope:{' '}
                {detail.scope || 'PUBLIC'} | bizTags: {detail.bizTags || '—'}
              </p>
              {detail.labels && Object.keys(detail.labels).length > 0 ? (
                <p>labels: {JSON.stringify(detail.labels)}</p>
              ) : null}
              <Table celled compact size='small'>
                <Table.Header>
                  <Table.Row>
                    <Table.HeaderCell>version</Table.HeaderCell>
                    <Table.HeaderCell>status</Table.HeaderCell>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {(detail.versions || []).map((v) => (
                    <Table.Row key={v.version}>
                      <Table.Cell>{v.version}</Table.Cell>
                      <Table.Cell>{v.status}</Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table>
            </>
          ) : null}
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setDetailOpen(false)}>{t('nacos.skills_close')}</Button>
        </Modal.Actions>
      </Modal>

      <Modal open={labelsOpen} onClose={() => setLabelsOpen(false)} size='large'>
        <Modal.Header>
          {t('nacos.prompts_labels_title')}: {labelsKey}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label='labels (JSON object)'
              language='json'
              enableJsonFormat
              minimap
              value={labelsText}
              originValue={labelsBaseline}
              onChange={(v) => setLabelsText(v)}
              height={360}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.prompts_labels_replace')}
                checked={labelsReplace}
                onChange={(e, { checked }) => setLabelsReplace(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setLabelsOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button primary onClick={saveLabels}>
            {t('nacos.skills_save')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={submitOpen} onClose={() => setSubmitOpen(false)}>
        <Modal.Header>
          {t('nacos.prompts_submit')}: {submitKey}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.skills_version_optional')}
              hint={t('nacos.skills_submit_hint')}
              value={submitVer}
              originValue={submitVerBaseline}
              onChange={(v) => setSubmitVer(v)}
              height={88}
            />
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setSubmitOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button primary onClick={doSubmit}>
            {t('nacos.prompts_submit')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Modal open={publishOpen} onClose={() => setPublishOpen(false)}>
        <Modal.Header>
          {t('nacos.prompts_publish')}: {pubKey}
        </Modal.Header>
        <Modal.Content>
          <Form>
            <SettingMonacoField
              label={t('nacos.skills_publish_version')}
              value={pubVer}
              originValue={pubVerBaseline}
              onChange={(v) => setPubVer(v)}
              height={88}
            />
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_update_latest')}
                checked={pubLatest}
                onChange={(e, { checked }) => setPubLatest(!!checked)}
              />
            </Form.Field>
            <Form.Field>
              <Checkbox
                label={t('nacos.skills_force_publish')}
                checked={pubForce}
                onChange={(e, { checked }) => setPubForce(!!checked)}
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setPublishOpen(false)}>{t('nacos.skills_close')}</Button>
          <Button positive onClick={doPublish}>
            {t('nacos.prompts_publish')}
          </Button>
        </Modal.Actions>
      </Modal>

      <Confirm
        open={deleteOpen}
        header={t('nacos.prompts_delete_confirm')}
        content={`${deleteKey} @ ${namespace}`}
        onCancel={() => {
          setDeleteOpen(false);
          setDeleteKey('');
        }}
        onConfirm={doDelete}
      />
    </div>
  );
};

export default NacosPromptsRegistry;
