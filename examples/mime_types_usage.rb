# frozen_string_literal: true

require "mime/types"

# Look up a MIME type by its content-type string. MIME::Types[str] returns the
# priority-sorted Array of matching MIME::Type value objects.
png = MIME::Types["image/png"].first
puts "content_type:        #{png.content_type}"        # => image/png
puts "media/sub:           #{png.media_type}/#{png.sub_type}"
puts "friendly:            #{png.friendly}"             # => Portable Network Graphics (PNG)
puts "extensions:          #{png.extensions.inspect}"  # => ["png"]
puts "preferred_extension: #{png.preferred_extension}" # => png
puts "encoding:            #{png.encoding}"            # => base64
puts "binary?:             #{png.binary?}"             # => true
puts "registered?:         #{png.registered?}"         # => true

# Look up by filename / extension with .type_for (aliased .of); the extension is
# matched case-insensitively and the result keeps the gem's priority ordering.
puts "report.pdf ->        #{MIME::Types.type_for('report.pdf').first.content_type}"
puts "archive.tar.gz ->    #{MIME::Types.of('archive.tar.gz').map(&:content_type).inspect}"

# The complete IANA registry is embedded — no gem, no network.
puts "registry size:       #{MIME::Types.count} type variants"
