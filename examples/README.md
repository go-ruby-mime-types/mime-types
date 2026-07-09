# Examples

Runnable pure-Ruby usage of `mime/types`, verified under the
[rbgo](https://github.com/go-embedded-ruby/ruby) interpreter.

```sh
rbgo examples/mime_types_usage.rb
```

| File | Shows |
| --- | --- |
| `mime_types_usage.rb` | `MIME::Types[str]` content-type lookup; `MIME::Type` readers (`content_type`, `media_type`, `sub_type`, `friendly`, `extensions`, `preferred_extension`, `encoding`) and predicates (`binary?`, `registered?`); `MIME::Types.type_for` / `.of` filename lookup; `MIME::Types.count` registry size |
