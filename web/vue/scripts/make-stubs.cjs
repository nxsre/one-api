#!/usr/bin/env node
'use strict';
const fs = require('fs');
const path = require('path');

const root = path.join(__dirname, '..', 'src');

// [relative path under src, display name]
const pages = [
  ['pages/Home/index.vue', 'Home'],
  ['pages/About/index.vue', 'About'],
  ['pages/NotFound/index.vue', 'NotFound'],
  ['pages/Channel/index.vue', 'Channel'],
  ['pages/Channel/EditChannel.vue', 'EditChannel'],
  ['pages/ModelCatalog/index.vue', 'ModelCatalog'],
  ['pages/Operation/index.vue', 'Operation'],
  ['pages/Routing/index.vue', 'Routing'],
  ['pages/Token/index.vue', 'Token'],
  ['pages/Token/EditToken.vue', 'EditToken'],
  ['pages/Redemption/index.vue', 'Redemption'],
  ['pages/Redemption/EditRedemption.vue', 'EditRedemption'],
  ['pages/User/index.vue', 'User'],
  ['pages/User/EditUser.vue', 'EditUser'],
  ['pages/User/AddUser.vue', 'AddUser'],
  ['pages/TenantUpgrades/index.vue', 'TenantUpgrades'],
  ['pages/TenantManagement/index.vue', 'TenantManagement'],
  ['pages/PlatformReports/index.vue', 'PlatformReports'],
  ['pages/TopUp/index.vue', 'TopUp'],
  ['pages/Log/index.vue', 'Log'],
  ['pages/Chat/index.vue', 'Chat'],
  ['pages/Dashboard/index.vue', 'Dashboard'],
  ['pages/Setting/index.vue', 'Setting'],
  ['pages/TenantConsole/index.vue', 'TenantConsole'],
  ['pages/TenantConsole/TenantReports.vue', 'TenantReports'],
  ['pages/TenantConsole/TenantUsers.vue', 'TenantUsers'],
  ['pages/TenantConsole/TenantEditUser.vue', 'TenantEditUser'],
  ['pages/TenantConsole/TenantChannels.vue', 'TenantChannels'],
  ['pages/TenantConsole/TenantEditChannel.vue', 'TenantEditChannel'],
  ['pages/TenantConsole/TenantUserTokens.vue', 'TenantUserTokens'],
  ['pages/TenantConsole/TenantEditToken.vue', 'TenantEditToken'],
  ['pages/Nacos/Namespaces.vue', 'NacosNamespaces'],
  ['pages/Nacos/NacosConsoleExternalOpen.vue', 'NacosConsoleExternalOpen'],
  ['pages/Nacos/CsConfigs.vue', 'NacosCsConfigs'],
  ['pages/Nacos/SkillsRegistry.vue', 'NacosSkillsRegistry'],
  ['pages/Nacos/AgentSpecsRegistry.vue', 'NacosAgentSpecsRegistry'],
  ['pages/Nacos/McpRegistry.vue', 'NacosMcpRegistry'],
  ['pages/Nacos/A2aRegistry.vue', 'NacosA2aRegistry'],
  ['pages/Nacos/PromptsRegistry.vue', 'NacosPromptsRegistry'],
  ['pages/Nacos/PipelinesRegistry.vue', 'NacosPipelinesRegistry'],
  ['pages/Nacos/NacosPermissions.vue', 'NacosPermissions'],
  ['components/LoginForm.vue', 'LoginForm'],
  ['components/RegisterForm.vue', 'RegisterForm'],
  ['components/PasswordResetForm.vue', 'PasswordResetForm'],
  ['components/PasswordResetConfirm.vue', 'PasswordResetConfirm'],
  ['components/GitHubOAuth.vue', 'GitHubOAuth'],
  ['components/LarkOAuth.vue', 'LarkOAuth'],
  ['components/TwoFASetting.vue', 'TwoFASetting'],
  ['components/TenantConsoleActingBar.vue', 'TenantConsoleActingBar'],
];

function stub(name) {
  return `<template>
  <div class="p-6">
    <a-card>
      <a-empty :description="'${name} — 待迁移 / pending migration'" />
    </a-card>
  </div>
</template>

<script setup>
// Placeholder for ${name}; replaced during page migration.
</script>
`;
}

let created = 0;
let skipped = 0;
for (const [rel, name] of pages) {
  const full = path.join(root, rel);
  if (fs.existsSync(full)) {
    skipped += 1;
    continue;
  }
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, stub(name));
  created += 1;
}
console.log(`stubs created: ${created}, skipped (existing): ${skipped}`);
