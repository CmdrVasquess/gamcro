<template>
<transition name="fade">
  <div v-if="status" id="stat" title="Status message. Click to close."
       @click="status=''">{{status}}</div>
</transition>
<h1><img src="logo.png" height="38" alt="G">amcro • Text Panel</h1>
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
body {
	background-color: #2c3e50;
    color: #F0B80F;
}
button {
	background-color: #5F83A7;
	color: #F0B80F;
	font-weight: bold;
	font-size: 102%;
	border: 2px solid #5F83A7;
	border-radius: .3em;
	box-shadow: 0 0 .4em 1px black;
	margin: .4em;
	padding: .4em 1em;
}
button:focus { border: 2px solid #A0BAD5; }
button:hover { color: #F7D87B; }
button:active { box-shadow: none; }
button:disabled {
	box-shadow: none;
	color: #A0BAD5;
	cursor: not-allowed;
}
label::after { content: ':'; }
label.before {
	font-weight: bold;
	margin-right: .3em;
}
input {
	background-color: #2c3e50;
	color: #F0B80F;
	font-size: 105%;
	border: none;
	border-bottom: 2px solid #5F83A7;
	padding: .2em .5em;
	padding-bottom: .1em;
	margin-bottom: .1em;
}
textarea {
	background-color: black;
	color: #F0B80F;
	font-size: 130%;
	border: 2px solid black;
	padding: .2em .5em;
}
textarea:focus { border: 2px solid #A0BAD5; }
#app {
    font-family: Avenir, Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    text-align: center;
}
h1 { margin: .5em 0; }
main { max-width: 60em; margin: auto; }
#top {
    display: flex;
    flex-flow: row-reverse wrap;
    justify-content: space-between;
    background-color: #2c3e50;
    position: sticky;
    top: 0;
    padding: 0.5em 0;
}
#stat {
    background-color: black;
    color: red;
    font-weight: bold;
    padding: .7em 0;
    margin: .5em 0;
    cursor: pointer;
}
#quick { display: inline-block; }
.message { margin-top: 1em; }
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
