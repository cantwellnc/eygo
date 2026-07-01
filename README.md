# eygo 
a go hosting of [EYG](https://github.com/crowdhailer/eyg-lang), an embeddable scripting language with managed effects + effect handlers. 

# why
i want to experiment with llms that can writing eyg programs. 

a harness can
- typecheck these / do other forms of static analysis on the generated code to ensure safety
- manage effect implementations outside of what the llm can access, or limit the effects it is even able to express
- ...

# why go

fast, single binary, large preexisting ecosystem of libraries for implementing effect handlers
