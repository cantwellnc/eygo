# eygo 
a go hosting of [EYG](https://github.com/crowdhailer/eyg-lang), an embeddable scripting language with managed effects + effect handlers. 

# why
i want to experiment with llms that can only write eyg programs. this constraint allows the harness to
- typecheck these / do other forms of static analysis on the generated code to ensure safety
- manage effect implementations outside of what the llm can access, or limit the effects it is even able to express
- ...
before any llm-generated code is ever run. 

# why go

fast, single binary, large preexisting ecosystem of libraries for implementing effect handlers
