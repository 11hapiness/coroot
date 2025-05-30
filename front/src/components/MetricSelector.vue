<template>
    <v-input :value="value" :rules="rules" dense hide-details @click="focus">
        <div class="d-flex align-center wrapper">
            <div v-if="wrapped && wrapped.prefix" class="grey--text">{{ wrapped.prefix }}</div>
            <div ref="cm" class="overflow-hidden" />
            <div v-if="wrapped && wrapped.suffix" class="grey--text">{{ wrapped.suffix }}</div>
        </div>
    </v-input>
</template>

<script>
import { PromQLExtension } from '@prometheus-io/codemirror-promql';
import { EditorState } from '@codemirror/state';
import { EditorView, keymap } from '@codemirror/view';
import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';

const themeSpec = {
    '&.cm-editor': {
        '&.cm-focused': {
            outline: 'none',
            outline_fallback: 'none',
        },
    },
    '.cm-scroller': {
        overflow: 'hidden',
        fontFamily: '"Roboto", sans-serif',
    },
    '.cm-completionIcon': {
        display: 'none',
    },
    '.cm-completionDetail': {
        display: 'none',
    },
    '.cm-tooltip.cm-completionInfo': {
        display: 'none',
    },
    '.cm-completionMatchedText': {
        textDecoration: 'none',
        fontWeight: 'bold',
    },
    '.cm-line': {
        padding: '0 4px',
    },
};

export default {
    props: {
        value: String,
        rules: Array,
        wrap: String,
    },

    view: null,

    watch: {
        value() {
            if (this.value !== this.view.state.doc.toString()) {
                this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: this.value } });
            }
        },
    },

    computed: {
        wrapped() {
            if (!this.wrap) {
                return null;
            }
            const parts = this.wrap.split('<input>', 2);
            if (parts.length === 0) {
                return null;
            }
            return { prefix: parts[0], suffix: parts[1] };
        },
    },

    mounted() {
        const enterKeymap = {
            key: 'Enter',
            run: (view) => {
                this.apply(view);
                return true;
            },
        };
        const promConf = {
            remote: {
                apiPrefix: this.$api.prom() + '/api/v1',
            },
        };
        const promQL = new PromQLExtension().setComplete(promConf);
        this.view = new EditorView({
            state: EditorState.create({
                doc: this.value,
                extensions: [
                    closeBrackets(),
                    autocompletion(),
                    keymap.of([enterKeymap, ...closeBracketsKeymap, ...completionKeymap]),
                    EditorView.theme(themeSpec, { dark: this.$vuetify.theme.dark }),
                    promQL.asExtension(),
                    EditorView.updateListener.of((update) => {
                        if (update.focusChanged) {
                            this.apply(update);
                        }
                    }),
                ],
            }),
            parent: this.$refs.cm,
        });
    },

    beforeDestroy() {
        this.view && this.view.destroy();
    },

    methods: {
        focus() {
            this.view && this.view.focus();
        },
        apply(v) {
            this.$emit('input', v.state.doc.toString());
        },
    },
};
</script>

<style scoped>
.wrapper {
    width: 100%;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 4px 8px;
}
</style>
