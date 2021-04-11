<template>
<h1>Gamcro – Game Macros</h1>
<div id="stat">{{status}}</div>
<main>
  <div id="top">
    <QuickText :typeMsg="typeMsg" :clipMsg="clipMsg"/>
    <button @click="addMsg()">New Text</button>
  </div>
  <Message v-for="(msg, idx) in msgs" :key="idx" :index="idx" v-model="msgs[idx]"
           :typeMsg="typeMsg" :clipMsg="clipMsg" :delMsg="delMsg"/>
</main>
</template>

<script>
import QuickText from './components/QuickText.vue'
import Message from './components/Message.vue'

export default {
    name: 'App',
    components: {
        QuickText,
        Message
    },
    data() {
        return {
            status: "",
            msgs: [""]
        }
    },
    methods: {
        addMsg() {
            this.msgs.push("");
            console.log("new msg");
        },
        delMsg(i) {
            if (this.msgs.length > 1) {
                this.msgs.splice(i, 1);
                return;
            }
            this.msgs = [""];
        },
        typeMsg(txt) { this.sendMsg("/keyboard/type", txt); },
        clipMsg(txt) { this.sendMsg("/clip", txt); },
        sendMsg(op, txt) {
            console.log("send", op, txt);
            this.status = "";
            let init = {
                method: "POST",
                headers: {'Content-Type': "text/plain"},
                body: txt
            };
            fetch(new Request(op, init))
                .then(resp => {
                    if (!resp.ok) {
                        this.status = resp.statusText;
                    }
                    this.qtype = "";
                    this.qclip = "";
                });
        }
    },
    mounted() {
        if (localStorage.msgs) {
            try {
                this.msgs = JSON.parse(localStorage.msgs);
            } catch(x) {
                console.log("load msgs from local storage: ", x);
                this.msgs = [""];
            }
        }
    },
    watch: {
        msgs: {
            handler() {
                localStorage.msgs = JSON.stringify(this.msgs);
            },
            deep: true
        }
    }
}
</script>

<style>
#app {
    font-family: Avenir, Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    text-align: center;
    color: #2c3e50;
}
main { max-width: 60em; margin: auto; }
#top {
    display: flex;
    flex-flow: row-reverse wrap;
    justify-content: space-between;
}
#stat { color: red; }
#quick { display: inline-block; }
.message { margin-top: 1em; }
</style>
