<template>
    <Views :loading="loading" :error="error">
        <template v-if="name" #subtitle>{{ name }}</template>

        <template v-if="node">
            <div v-if="node.status === 'unknown'" class="text-center">
                This node is present in the Kubernetes cluster, but it seems that coroot-node-agent is not installed.
            </div>
            <Dashboard v-else :name="name" :widgets="node.widgets" class="mt-3" />
        </template>
        <NoData v-else-if="!loading && !error" />
    </Views>
</template>

<script>
import Views from '@/views/Views.vue';
import Dashboard from '../components/Dashboard';
import NoData from '../components/NoData';

export default {
    props: {
        name: String,
    },

    components: { Views, Dashboard, NoData },

    data() {
        return {
            node: null,
            loading: false,
            error: '',
        };
    },

    mounted() {
        this.get();
        this.$events.watch(this, this.get, 'refresh');
    },

    watch: {
        name() {
            this.node = null;
            this.get();
        },
    },

    methods: {
        get() {
            this.loading = true;
            this.$api.getNode(this.name, (data, error) => {
                this.loading = false;
                if (error) {
                    this.error = error;
                    return;
                }
                this.node = data;
            });
        },
    },
};
</script>

<style scoped></style>
