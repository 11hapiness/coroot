<template>
    <div>
        <v-alert v-if="error" color="red" icon="mdi-alert-octagon-outline" outlined text class="mt-2">
            {{ error }}
        </v-alert>
        <v-alert v-if="disabled" color="info" outlined text>
            Single Sign-On through SAML is available only in Coroot Enterprise (from $1 per CPU core/month).
            <a href="https://coroot.com/account" target="_blank" class="font-weight-bold">Start</a> your free trial today.
        </v-alert>
        <v-alert v-if="readonly" color="primary" outlined text>
            Single Sing-On is configured through the config and cannot be modified via the UI.
        </v-alert>
        <v-simple-table v-if="status !== 403" dense class="params">
            <tbody>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Status</td>
                    <td>
                        <div v-if="enabled">
                            <v-icon color="success" class="mr-1" size="20">mdi-check-circle</v-icon>
                            Enabled
                        </div>
                        <div v-else>Disabled</div>
                    </td>
                </tr>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Identity Provider:</td>
                    <td>
                        <span v-if="provider" style="vertical-align: middle">{{ provider }}</span>
                        <input ref="file" type="file" accept=".xml" @change="upload" class="d-none" />
                        <v-btn v-if="!provider" color="primary" small :disabled="disabled || loading || readonly" @click="$refs.file.click()">
                            Upload Identity Provider Metadata XML
                        </v-btn>
                        <v-btn v-else :disabled="disabled || loading || readonly" small icon @click="$refs.file.click()">
                            <v-icon small>mdi-pencil</v-icon>
                        </v-btn>
                    </td>
                </tr>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Service Provider Issuer / Identity ID:</td>
                    <td>{{ asc_url }} <CopyButton :text="asc_url" :disabled="disabled" /></td>
                </tr>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Service Provider ACS URL / Single Sign On URL:</td>
                    <td>{{ asc_url }} <CopyButton :text="asc_url" :disabled="disabled" /></td>
                </tr>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Attribute mapping:</td>
                    <td>
                        Coroot expects to receive the <b>Email</b>, <b>FirstName</b>, and <b>LastName</b> attributes.
                        <br />
                        Please configure Attribute Mapping on your Identity Provider's side.
                    </td>
                </tr>
                <tr>
                    <td class="font-weight-medium text-no-wrap">Default role:</td>
                    <td>
                        <v-select
                            v-model="default_role"
                            :items="roles"
                            :disabled="disabled || readonly"
                            outlined
                            dense
                            :menu-props="{ offsetY: true }"
                            :rules="[$validators.notEmpty]"
                            hide-details
                            class="roles"
                        />
                    </td>
                </tr>
            </tbody>
        </v-simple-table>
        <div v-if="status !== 403" class="d-flex mt-2" style="gap: 8px">
            <v-btn color="primary" small :disabled="disabled || loading || readonly || !provider" @click="save">
                Save <template v-if="!enabled">and Enable</template>
            </v-btn>
            <v-btn v-if="enabled" color="error" small :disabled="disabled || loading || readonly" @click="disable">Disable</v-btn>
        </div>
    </div>
</template>

<script>
import CopyButton from '@/components/CopyButton.vue';

export default {
    components: { CopyButton },
    computed: {
        asc_url() {
            return location.origin + this.$coroot.base_path + 'sso/saml';
        },
    },

    data() {
        return {
            disabled: this.$coroot.edition !== 'Enterprise',
            readonly: false,
            loading: false,
            error: '',
            status: undefined,
            enabled: false,
            default_role: '',
            provider: '',
            roles: [],
        };
    },

    mounted() {
        this.$events.watch(this, this.get, 'roles');
        this.get();
    },

    methods: {
        get() {
            this.loading = true;
            this.error = '';
            this.status = undefined;
            this.$api.sso(null, (data, error, status) => {
                this.loading = false;
                if (error) {
                    this.error = error;
                    this.status = status;
                    return;
                }
                this.readonly = data.readonly;
                this.enabled = data.enabled;
                this.default_role = data.default_role;
                this.provider = data.provider;
                this.roles = data.roles || [];
            });
        },
        post(action, metadata) {
            this.loading = true;
            this.error = '';
            this.status = undefined;
            const form = {
                action,
                default_role: this.default_role,
                saml: metadata ? { metadata } : undefined,
            };
            this.$api.sso(form, (data, error, status) => {
                this.loading = false;
                if (error) {
                    this.error = error;
                    this.status = status;
                    return;
                }
                this.get();
            });
        },
        save() {
            this.post('save');
        },
        disable() {
            this.post('disable');
        },
        upload(e) {
            const file = e.target.files[0];
            e.target.value = '';
            if (!file) {
                return;
            }
            file.text().then((v) => {
                this.post('upload', v);
            });
        },
    },
};
</script>

<style scoped>
.params:deep(td) {
    padding: 4px 16px !important;
}
.roles {
    max-width: 20ch;
}
.roles:deep(.v-input__slot) {
    min-height: initial !important;
    height: 2rem !important;
    padding: 0 8px !important;
}
.roles:deep(.v-input__append-inner) {
    margin-top: 4px !important;
}
</style>
